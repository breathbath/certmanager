package logging

import (
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
)

type Config struct {
	LogLevel string `envconfig:"LOGGING_LEVEL"`
	LogKey   string `envconfig:"LOGGING_KEY"`
}

func LoadConfig() (cfg *Config, err error) {
	cfg = new(Config)
	err = envconfig.Process("logging", cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load db config")
	}

	return cfg, nil
}
