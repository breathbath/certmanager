package certmanager

import (
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
	"time"
)

type Config struct {
	SecretName       string        `envconfig:"SECRET_NAME" required:"true"`
	TargetNamespaces []string      `envconfig:"TARGET_NAMESPACES" required:"true"`
	RunInterval      time.Duration `envconfig:"RUN_INTERVAL" default:"5m"`
	ChallengePath    string        `envconfig:"CHALLENGE_PATH" required:"true"`
	CertEmail        string        `envconfig:"CERT_EMAIL" required:"true"`
	CertDomain       string        `envconfig:"CERT_DOMAIN" required:"true"`
	CertIssTimeout   time.Duration `envconfig:"ISSUE_TIMEOUT" default:"20m"`
}

func LoadConfig() (cfg *Config, err error) {
	cfg = new(Config)
	err = envconfig.Process("certmanager", cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load db config")
	}

	return cfg, nil
}
