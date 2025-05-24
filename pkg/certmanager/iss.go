package certmanager

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"os"
	"path"
)

// CustomProvider implements http01.Provider interface
type CustomProvider struct {
	cfg          *Config
	currentToken string
}

func (p *CustomProvider) Present(_, token, keyAuth string) error {
	filePath := path.Join(p.cfg.ChallengePath, token)
	logrus.Infof("Presenting the challenge for token: %s", token)

	err := os.WriteFile(filePath, []byte(keyAuth), 0644)
	if err != nil {
		logrus.Errorf("Error writing challenge file for token %s: %v", token, err)
		return errors.Wrap(err, "Failed to write challenge file")
	}

	logrus.Debugf("Successfully wrote challenge file for token: %s", token)
	p.currentToken = token

	return nil
}

func (p *CustomProvider) CleanUp(_, token, _ string) error {
	logrus.Infof("Cleaning up the challenge for token: %s", token)

	err := p.delete(token)
	if err != nil {
		logrus.Errorf("Error cleaning up challenge file for token %s: %v", token, err)
		return err
	}

	if p.currentToken == token {
		logrus.Debug("Clearing current token in CustomProvider")
		p.currentToken = ""
	}

	return nil
}

func (p *CustomProvider) delete(token string) error {
	filePath := path.Join(p.cfg.ChallengePath, token)
	logrus.Infof("Deleting challenge file: %s", filePath)

	err := os.Remove(filePath)
	if err != nil {
		logrus.Errorf("Error deleting challenge file for token %s: %v", token, err)
		return errors.Wrap(err, "Failed to remove challenge file")
	}

	logrus.Debugf("Successfully deleted challenge file for token: %s", token)
	return nil
}

func (p *CustomProvider) Cleanup() error {
	if p.currentToken == "" {
		return nil
	}

	logrus.Debugf("Performing cleanup for currentToken: %s", p.currentToken)
	return p.delete(p.currentToken)
}

// User implements lego.User interface
type User struct {
	Email        string
	Registration *registration.Resource
	Key          crypto.PrivateKey
}

func (u *User) GetEmail() string {
	logrus.Debugf("Getting email for user: %s", u.Email)
	return u.Email
}

func (u *User) GetRegistration() *registration.Resource {
	logrus.Debugf("Getting registration data for user with email: %s", u.Email)
	return u.Registration
}

func (u *User) GetPrivateKey() crypto.PrivateKey {
	logrus.Debug("Getting private key for user")
	return u.Key
}

func (cm *CertManager) Issue() (cert, pk []byte, err error) {
	logrus.Info("Starting certificate issuance process")

	userKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		logrus.Errorf("Error generating private key: %v", err)
		return nil, nil, errors.Wrap(err, "Failed to generate private key")
	}
	logrus.Debug("Successfully generated private key")

	user := &User{
		Email: cm.cfg.CertEmail,
		Key:   userKey,
	}
	logrus.Infof("Defined ACME user with email: %s", cm.cfg.CertEmail)

	config := lego.NewConfig(user)
	config.CADirURL = lego.LEDirectoryStaging

	client, err := lego.NewClient(config)
	if err != nil {
		logrus.Errorf("Error creating ACME client: %v", err)
		return nil, nil, errors.Wrap(err, "Failed to create ACME client")
	}

	provider := &CustomProvider{cfg: cm.cfg}
	if err := client.Challenge.SetHTTP01Provider(provider); err != nil {
		logrus.Errorf("Error setting HTTP-01 provider: %v", err)
		return nil, nil, errors.Wrap(err, "Failed to set HTTP-01 provider")
	}

	reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
	if err != nil {
		logrus.Errorf("Error registering user: %v", err)
		return nil, nil, errors.Wrap(err, "Failed to register user")
	}
	user.Registration = reg

	request := certificate.ObtainRequest{
		Domains: []string{cm.cfg.CertDomain},
		Bundle:  true,
	}

	certRes, err := cm.obtain(request, client)
	if err != nil {
		logrus.Errorf("Error obtaining certificate: %v", err)
		if err2 := provider.Cleanup(); err2 != nil {
			logrus.Errorf("Cleanup failed: %v", err2)
		}
		return nil, nil, errors.Wrap(err, "Failed to obtain certificate")
	}

	logrus.Info(
		"Certificate successfully obtained, CertURL: %s, CertStableURL: %s, Domain: %s",
		certRes.CertURL,
		certRes.CertStableURL,
		certRes.Domain,
	)

	return certRes.PrivateKey, certRes.Certificate, nil
}

func (cm *CertManager) obtain(request certificate.ObtainRequest, client *lego.Client) (cert *certificate.Resource, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), cm.cfg.CertIssTimeout)
	defer cancel()

	logrus.Infof(
		"Requesting certificate for domain: %s with timeout: %v",
		cm.cfg.CertDomain,
		cm.cfg.CertIssTimeout,
	)

	done := make(chan struct{})
	var certRes *certificate.Resource
	var obtainErr error

	go func() {
		certRes, obtainErr = client.Certificate.Obtain(request)
		close(done)
	}()

	select {
	case <-done:
		if obtainErr != nil {
			return nil, obtainErr
		}

		return certRes, nil
	case <-ctx.Done():
		logrus.Errorf("Obtain operation timed out after %v", cm.cfg.CertIssTimeout)
		return nil, fmt.Errorf("Obtain operation timed out after %v", cm.cfg.CertIssTimeout)
	}
}
