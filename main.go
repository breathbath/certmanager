package main

import (
	"fmt"
	"log"

	"github.com/breathbath/certmanager/internal/config"
	"github.com/breathbath/certmanager/internal/k8s"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	clientset, err := k8s.NewClient()
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	sm := k8s.NewSecretManager(clientset)
	fmt.Println("Ensuring dummy secrets...")
	sm.EnsureDummySecret(cfg.SecretName, cfg.TargetNamespaces)
}
