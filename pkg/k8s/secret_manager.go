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
	secretName string,
	namespaces []string,
	issue func() (certPEM, keyPEM []byte, err error),
) error {
	minValidity := 1 * time.Hour

	for _, ns := range namespaces {
		ns = strings.TrimSpace(ns)

		secret, err := sm.clientset.CoreV1().Secrets(ns).Get(ctx, secretName, metav1.GetOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return errors.Wrapf(err, "failed to get secret %s/%s", ns, secretName)
		}

		isSecretFound := !apierrors.IsNotFound(err)

		logrus.Infof("secret %s/%s found: %v", ns, secretName, isSecretFound)

		if isSecretFound && sm.IsCertValid(secret, minValidity) {
			logrus.Infof("secret %s/%s already exists and is valid", ns, secretName)
			continue
		}

		logrus.Infof("secret %s/%s does not exist or is not valid, generating a new one", ns, secretName)

		certPEM, keyPEM, err := issue()
		if err != nil {
			return errors.Wrapf(err, "failed to generate cert for %s", ns)
		}

		secretData := map[string][]byte{
			"tls.crt": certPEM,
			"tls.key": keyPEM,
		}

		tlsSecret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: ns,
			},
			Type: v1.SecretTypeTLS,
			Data: secretData,
		}

		if isSecretFound {
			// Update existing secret with retry on conflict
			for i := 0; i < 3; i++ {
				secret.Data = secretData
				secret.Type = v1.SecretTypeTLS

				if _, err := sm.clientset.CoreV1().Secrets(ns).Update(context.TODO(), secret, metav1.UpdateOptions{}); err != nil {
					if apierrors.IsConflict(err) {
						secret, err = sm.clientset.CoreV1().Secrets(ns).Get(context.TODO(), secretName, metav1.GetOptions{})
						if err != nil {
							return errors.Wrapf(err, "failed to get secret %s/%s on conflict retry", ns, secretName)
						}
						continue
					}
					return errors.Wrapf(err, "failed to update secret %s/%s", ns, secretName)
				}

				logrus.Infof("updated secret %s/%s", ns, secretName)

				break
			}
		} else {
			if _, err := sm.clientset.CoreV1().Secrets(ns).Create(context.TODO(), tlsSecret, metav1.CreateOptions{}); err != nil {
				return errors.Wrapf(err, "failed to create secret %s/%s", ns, secretName)
			}
			logrus.Infof("created secret %s/%s", ns, secretName)
		}
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
