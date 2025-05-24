package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/breathbath/certmanager/internal/config"
	"github.com/breathbath/certmanager/internal/k8s"
)

func runPeriodically(ctx context.Context, cfg *config.Config, sm *k8s.SecretManager) {
	ticker := time.NewTicker(cfg.RunInterval)
	defer ticker.Stop()

	log.Printf("Starting periodic secret checks every %s\n", cfg.RunInterval)

	for {
		select {
		case <-ticker.C:
			log.Println("Running periodic secret check...")
			err := execute(cfg, sm)
			if err != nil {
				log.Println(err)
			} else {
				log.Println("Secret check completed successfully")
			}
		case <-ctx.Done():
			log.Println("Shutting down...")
			return
		}
	}
}

func execute(cfg *config.Config, sm *k8s.SecretManager) error {
	return sm.EnsureTLSSecret(cfg.SecretName, cfg.TargetNamespaces)
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	clientset, err := k8s.NewClient()
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	sm := k8s.NewSecretManager(clientset)

	// Initial execution
	log.Println("Running initial secret check...")
	err = execute(cfg, sm)
	if err != nil {
		log.Fatal(err)
	}

	go runPeriodically(ctx, cfg, sm)

	sig := <-sigs
	log.Printf("Received signal %s, shutting down...\n", sig)
}
