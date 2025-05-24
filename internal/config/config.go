package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	SecretName       string
	TargetNamespaces []string
	RunInterval      time.Duration
}

func LoadConfig() (*Config, error) {
	secretName := os.Getenv("SECRET_NAME")
	namespaces := os.Getenv("TARGET_NAMESPACES")

	if secretName == "" || namespaces == "" {
		return nil, fmt.Errorf("missing SECRET_NAME or TARGET_NAMESPACES")
	}

	intervalStr := os.Getenv("CHECK_INTERVAL")

	var runInterval time.Duration
	var err error
	if intervalStr == "" {
		runInterval = time.Minute * 5
	} else {
		runInterval, err = time.ParseDuration(intervalStr)
		if err != nil {
			return nil, fmt.Errorf("Invalid CHECK_INTERVAL: %v", err)
		}
	}

	return &Config{
		SecretName:       secretName,
		TargetNamespaces: strings.Split(namespaces, ","),
		RunInterval:      runInterval,
	}, nil
}
