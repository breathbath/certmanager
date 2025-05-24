package certmanager

import (
	"context"
	"github.com/breathbath/certmanager/pkg/k8s"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"time"
)

type CertManager struct {
	cfg               *Config
	kubeSecretManager *k8s.SecretManager
}

func NewCertManager() (*CertManager, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := k8s.NewClient()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create Kubernetes client")
	}

	sm := k8s.NewSecretManager(clientset)

	return &CertManager{
		cfg:               cfg,
		kubeSecretManager: sm,
	}, nil
}

func (cm *CertManager) Start(mainCtx context.Context) error {
	logrus.Info("Running initial secret check...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := cm.RunOnce(ctx)
	if err != nil {
		return err
	}

	go cm.RunPeriodically(mainCtx)

	return nil
}

func (cm *CertManager) RunPeriodically(mainCtx context.Context) {
	ticker := time.NewTicker(cm.cfg.RunInterval)
	defer ticker.Stop()

	logrus.Infof("Starting periodic secret checks every %s", cm.cfg.RunInterval)

	for {
		select {
		case <-ticker.C:
			logrus.Info("Running periodic secret check...")
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := cm.RunOnce(ctx)
			if err != nil {
				logrus.Error(err)
			} else {
				logrus.Info("Secret check completed successfully")
			}
		case <-mainCtx.Done():
			return
		}
	}
}

func (cm *CertManager) RunOnce(ctx context.Context) error {
	return cm.kubeSecretManager.EnsureTLSSecret(ctx, cm.cfg.SecretName, cm.cfg.TargetNamespaces, cm.Issue)
}
