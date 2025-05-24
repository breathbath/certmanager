package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	SecretName       string
	TargetNamespaces []string
}

func LoadConfig() (*Config, error) {
	secretName := os.Getenv("SECRET_NAME")
	namespaces := os.Getenv("TARGET_NAMESPACES")

	if secretName == "" || namespaces == "" {
		return nil, fmt.Errorf("missing SECRET_NAME or TARGET_NAMESPACES")
	}

	return &Config{
		SecretName:       secretName,
		TargetNamespaces: strings.Split(namespaces, ","),
	}, nil
}
