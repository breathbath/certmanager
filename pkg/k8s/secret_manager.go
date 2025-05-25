package k8s

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

type SecretManager struct {
	clientset *kubernetes.Clientset
}

func NewSecretManager(clientset *kubernetes.Clientset) *SecretManager {
	return &SecretManager{clientset: clientset}
}

func (sm *SecretManager) EnsureTLSSecret(
	ctx context.Context,
	namespace, domain, secretName, email string,
	issue func(mail, domain string) (certPEM, keyPEM []byte, err error),
) error {
	if namespace == "" || domain == "" || secretName == "" || email == "" {
		return errors.New("namespace, domain, secretName and email must be set")
	}

	minValidity := 1 * time.Hour

	namespace = strings.TrimSpace(namespace)

	secret, err := sm.clientset.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return errors.Wrapf(err, "failed to request secret %s/%s from k8s api", namespace, secret)
	}

	isSecretFound := !apierrors.IsNotFound(err)

	logrus.Infof("secret %s/%s found: %v", namespace, secret, isSecretFound)

	if isSecretFound && sm.IsCertValid(secret, minValidity) {
		logrus.Infof("secret %s/%s already exists and is valid", namespace, secret)
		return nil
	}

	logrus.Infof("secret %s/%s does not exist or is not valid, generating a new one", namespace, secret)

	certPEM, keyPEM, err := issue(email, domain)
	if err != nil {
		return errors.Wrapf(err, "failed to generate cert for %s", domain)
	}

	secretData := map[string][]byte{
		"tls.crt": certPEM,
		"tls.key": keyPEM,
	}

	tlsSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
		Type: v1.SecretTypeTLS,
		Data: secretData,
	}

	if isSecretFound {
		// Update existing secret with retry on conflict
		for i := 0; i < 3; i++ {
			secret.Data = secretData
			secret.Type = v1.SecretTypeTLS

			if _, err := sm.clientset.CoreV1().Secrets(namespace).Update(context.TODO(), secret, metav1.UpdateOptions{}); err != nil {
				if apierrors.IsConflict(err) {
					secret, err = sm.clientset.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
					if err != nil {
						return errors.Wrapf(err, "failed to get secret %s/%s on conflict retry", namespace, secretName)
					}
					continue
				}
				return errors.Wrapf(err, "failed to update secret %s/%s", namespace, secretName)
			}

			logrus.Infof("updated secret %s/%s", namespace, secretName)

			break
		}
	} else {
		if _, err := sm.clientset.CoreV1().Secrets(namespace).Create(context.TODO(), tlsSecret, metav1.CreateOptions{}); err != nil {
			return errors.Wrapf(err, "failed to create secret %s/%s", namespace, secretName)
		}
		logrus.Infof("created secret %s/%s", namespace, secretName)
	}

	return nil
}

func (sm *SecretManager) IsCertValid(secret *v1.Secret, minValidity time.Duration) bool {
	crtData, ok := secret.Data["tls.crt"]
	if !ok {
		return false
	}

	block, _ := pem.Decode(crtData)
	if block == nil {
		return false
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return false
	}

	return cert.NotAfter.After(time.Now().Add(minValidity))
}
