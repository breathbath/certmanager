package certmanager

import (
	"encoding/json"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"os"
	"time"
)

type CertTask struct {
	Namespace string `json:"Namespace"`
	Domain    string `json:"Domain"`
	Secret    string `json:"Secret"`
	Email     string `json:"Email"`
}

type Config struct {
	RunInterval    time.Duration `envconfig:"RUN_INTERVAL" default:"5m"`
	ChallengePath  string        `envconfig:"CHALLENGE_PATH" required:"true"`
	CertIssTimeout time.Duration `envconfig:"ISSUE_TIMEOUT" default:"20m"`
	ConfigPath     string        `envconfig:"CONFIG_PATH" requited:"true"`
	CertTasks      []CertTask
}

func (c *Config) loadTasks() error {
	file, err := os.ReadFile(c.ConfigPath)
	if err != nil {
		return errors.Wrapf(err, "failed to read config file %s", c.ConfigPath)
	}

	tasls := []CertTask{}
	err = json.Unmarshal(file, &tasls)
	if err != nil {
		return errors.Wrapf(err, "failed to unmarshal config file %s: %s into ConfigTasks", c.ConfigPath, string(file))
	}

	for _, task := range tasls {
		if task.Namespace == "" {
			return errors.Errorf("namespace is empty in task %+v", task)
		}
		if task.Domain == "" {
			return errors.Errorf("domain is empty in task %+v", task)
		}
		if task.Secret == "" {
			return errors.Errorf("secret is empty in task %+v", task)
		}
		if task.Email == "" {
			return errors.Errorf("Email is empty in task %+v", task)
		}
	}

	c.CertTasks = tasls

	return nil
}

func LoadConfig() (cfg *Config, err error) {
	cfg = new(Config)
	err = envconfig.Process("certmanager", cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load db config")
	}

	err = cfg.loadTasks()
	if err != nil {
		return nil, err
	}

	logrus.Infof("loaded certmanager config: %+v", cfg)

	return cfg, nil
}
