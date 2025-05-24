package challenge

import (
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
)

type Config struct {
	Port          int    `envconfig:"PORT" default:"8080"`
	ChallengePath string `envconfig:"PATH" required:"true"`
}

func LoadConfig() (cfg *Config, err error) {
	cfg = new(Config)
	err = envconfig.Process("challenge", cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load config")
	}

	return cfg, nil
}
