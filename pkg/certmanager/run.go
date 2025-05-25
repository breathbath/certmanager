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

func (cm *CertManager) RunPeriodically(mainCtx context.Context) {
	logrus.Infof(
		"Waiting for initial delay of %v before starting periodic checks",
		cm.cfg.InitialDelay,
	)

	// Wait for initial delay
	select {
	case <-time.After(cm.cfg.InitialDelay):
		logrus.Info("Running the initial secret check after delay...")
		cm.runTasks()
	case <-mainCtx.Done():
		return
	}

	// Start periodic loop
	ticker := time.NewTicker(cm.cfg.RunInterval)
	defer ticker.Stop()

	logrus.Infof("Starting periodic secret checks every %s", cm.cfg.RunInterval)
	for {
		select {
		case <-ticker.C:
			logrus.Info("Running periodic secret check...")
			cm.runTasks()
		case <-mainCtx.Done():
			return
		}
	}
}

func (cm *CertManager) runTasks() {
	for _, task := range cm.cfg.CertTasks {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := cm.kubeSecretManager.EnsureTLSSecret(
			ctx,
			task.Namespace,
			task.Domain,
			task.Secret,
			task.Email,
			cm.Issue,
		)
		if err != nil {
			logrus.Error(err)
		} else {
			logrus.Info("Secret check completed successfully")
		}
	}
}
