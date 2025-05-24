package k8s

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/breathbath/certmanager/internal/tlsutil"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
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

func (sm *SecretManager) EnsureTLSSecret(secretName string, namespaces []string) error {
	minValidity := 7 * 24 * time.Hour

	for _, ns := range namespaces {
		ns = strings.TrimSpace(ns)

		secret, err := sm.clientset.CoreV1().Secrets(ns).Get(context.TODO(), secretName, metav1.GetOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to get secret %s/%s: %w", ns, secretName, err)
		}

		isSecretFound := !apierrors.IsNotFound(err)

		log.Printf("secret %s/%s found: %v\n", ns, secretName, isSecretFound)

		if isSecretFound && sm.IsCertValid(secret, minValidity) {
			log.Printf("secret %s/%s already exists and is valid\n", ns, secretName)
			continue
		}

		log.Printf("secret %s/%s does not exist or is not valid, generating a new one\n", ns, secretName)

		certPEM, keyPEM, err := tlsutil.GenerateSelfSignedCert()
		if err != nil {
			return fmt.Errorf("failed to generate cert for %s: %w", ns, err)
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
							return fmt.Errorf("failed to get secret %s/%s on conflict retry: %w", ns, secretName, err)
						}
						continue
					}
					return fmt.Errorf("failed to update secret %s/%s: %w", ns, secretName, err)
				}
				log.Printf("updated secret %s/%s\n", ns, secretName)
				break
			}
		} else {
			if _, err := sm.clientset.CoreV1().Secrets(ns).Create(context.TODO(), tlsSecret, metav1.CreateOptions{}); err != nil {
				return fmt.Errorf("failed to create secret %s/%s: %w", ns, secretName, err)
			}
			log.Printf("created secret %s/%s\n", ns, secretName)
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
