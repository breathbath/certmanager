package k8s

import (
	"context"
	"fmt"
	"log"
	"strings"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type SecretManager struct {
	clientset *kubernetes.Clientset
}

func NewSecretManager(clientset *kubernetes.Clientset) *SecretManager {
	return &SecretManager{clientset: clientset}
}

func (sm *SecretManager) EnsureDummySecret(secretName string, namespaces []string) {
	for _, ns := range namespaces {
		ns = strings.TrimSpace(ns)
		fmt.Printf("Processing namespace: %s\n", ns)

		_, err := sm.clientset.CoreV1().Secrets(ns).Get(context.TODO(), secretName, metav1.GetOptions{})
		if err == nil {
			fmt.Printf("Secret %s already exists in namespace %s\n", secretName, ns)
			continue
		}

		secret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: secretName,
			},
			Type: v1.SecretTypeOpaque,
			Data: map[string][]byte{
				"dummy": []byte("dummydata"),
			},
		}

		_, err = sm.clientset.CoreV1().Secrets(ns).Create(context.TODO(), secret, metav1.CreateOptions{})
		if err != nil {
			log.Printf("Error creating secret in namespace %s: %v\n", ns, err)
		} else {
			fmt.Printf("Created dummy secret %s in namespace %s\n", secretName, ns)
		}
	}
}
