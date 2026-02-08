package k8s

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml"
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
	backupPath string,
	issue func(mail, domain string) (certPEM, keyPEM []byte, err error),
) error {
	if namespace == "" || domain == "" || secretName == "" || email == "" {
		return errors.New("namespace, domain, secretName and email must be set")
	}

	minValidity := 1 * time.Hour

	namespace = strings.TrimSpace(namespace)

	secret, err := sm.clientset.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return errors.Wrapf(err, "failed to request secret %s/%s from k8s api", namespace, secretName)
	}

	isSecretFound := !apierrors.IsNotFound(err)

	logrus.Infof("secret %s/%s found: %v", namespace, secretName, isSecretFound)

	if isSecretFound && sm.IsCertValid(secret, minValidity) {
		logrus.Infof("secret %s/%s already exists and is valid", namespace, secretName)
		return nil
	}

	logrus.Infof("secret %s/%s does not exist or is not valid, generating a new one", namespace, secretName)

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
		updated := false
		var lastErr error
		for i := 0; i < 3; i++ {
			secret.Data = secretData
			secret.Type = v1.SecretTypeTLS

			if _, err := sm.clientset.CoreV1().Secrets(namespace).Update(context.TODO(), secret, metav1.UpdateOptions{}); err != nil {
				if apierrors.IsConflict(err) {
					secret, err = sm.clientset.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
					if err != nil {
						lastErr = errors.Wrapf(err, "failed to get secret %s/%s on conflict retry", namespace, secretName)
						break
					}
					continue
				}
				lastErr = errors.Wrapf(err, "failed to update secret %s/%s", namespace, secretName)
				break
			}

			logrus.Infof("updated secret %s/%s", namespace, secretName)

			updated = true
			lastErr = nil
			break
		}
		if !updated {
			if lastErr == nil {
				lastErr = errors.Errorf("failed to update secret %s/%s after retries", namespace, secretName)
			}
			sm.backupOnFailure(backupPath, namespace, secretName, domain, certPEM, keyPEM)
			return lastErr
		}
	} else {
		if _, err := sm.clientset.CoreV1().Secrets(namespace).Create(context.TODO(), tlsSecret, metav1.CreateOptions{}); err != nil {
			sm.backupOnFailure(backupPath, namespace, secretName, domain, certPEM, keyPEM)
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

func (sm *SecretManager) backupOnFailure(
	backupPath, namespace, secretName, domain string,
	certPEM, keyPEM []byte,
) {
	if strings.TrimSpace(backupPath) == "" {
		return
	}

	backupFilePath, err := sm.backupSecretData(backupPath, namespace, secretName, domain, certPEM, keyPEM)
	if err != nil {
		logrus.WithError(err).Warn("failed to back up certificate data after secret install failure")
		return
	}

	logrus.Infof("backed up certificate data to %s", backupFilePath)
}

func (sm *SecretManager) backupSecretData(
	backupPath, namespace, secretName, domain string,
	certPEM, keyPEM []byte,
) (string, error) {
	safeNamespace := sanitizeName(namespace)
	safeSecret := sanitizeName(secretName)
	safeDomain := sanitizeName(domain)
	timestamp := time.Now().UTC().Format("20060102T150405Z")

	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return "", errors.Wrapf(err, "failed to create backup path %s", backupPath)
	}
	info, err := os.Stat(backupPath)
	if err != nil {
		return "", errors.Wrapf(err, "backup path %s is not accessible", backupPath)
	}
	if !info.IsDir() {
		return "", errors.Errorf("backup path %s is not a directory", backupPath)
	}

	backupFile := fmt.Sprintf("%s_%s_%s_%s.yaml", safeNamespace, safeSecret, safeDomain, timestamp)
	secretPath := filepath.Join(backupPath, backupFile)
	secretYAML, err := buildTLSSecretYAML(namespace, secretName, certPEM, keyPEM)
	if err != nil {
		return "", errors.Wrap(err, "failed to build secret manifest")
	}
	if err := os.WriteFile(secretPath, []byte(secretYAML), 0644); err != nil {
		return "", errors.Wrapf(err, "failed to write secret backup %s", secretPath)
	}

	return secretPath, nil
}

func buildTLSSecretYAML(namespace, secretName string, certPEM, keyPEM []byte) (string, error) {
	secret := &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
		Type: v1.SecretTypeTLS,
		Data: map[string][]byte{
			"tls.crt": certPEM,
			"tls.key": keyPEM,
		},
	}

	out, err := yaml.Marshal(secret)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal secret yaml")
	}

	return string(out), nil
}

func sanitizeName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}

	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= 'A' && r <= 'Z':
			return r
		case r >= '0' && r <= '9':
			return r
		default:
			return '_'
		}
	}, value)
}
