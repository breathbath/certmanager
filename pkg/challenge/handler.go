package challenge

import (
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"path"
	"strings"
)

type Handler struct {
	cfg *Config
}

func NewHandler(cfg *Config) *Handler {
	return &Handler{
		cfg: cfg,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logrus.Infof("Received request: %s %s", r.Method, r.URL.Path)

	prefix := "/.well-known/acme-challenge/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		logrus.Errorf("Invalid request path, does not match prefix: %s", r.URL.Path)
		http.NotFound(w, r)
		return
	}

	logrus.Debugf("Request passed prefix validation: %s", r.URL.Path)

	token := strings.TrimPrefix(r.URL.Path, prefix)
	if token == "" {
		logrus.Errorf("Token is missing from path: %s", r.URL.Path)
		http.NotFound(w, r)
		return
	}
	logrus.Infof("Token extracted from request path: %s", token)

	files, err := os.ReadDir(h.cfg.ChallengePath)
	if err != nil {
		logrus.Errorf("Failed to read challenge directory '%s': %s", h.cfg.ChallengePath, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var matchedFile string
	for _, f := range files {
		logrus.Debugf("Checking file: %s", f.Name())
		if !f.IsDir() && f.Name() == token {
			matchedFile = f.Name()
			logrus.Infof("Matching challenge file found: %s", matchedFile)
			break
		}
	}
	if matchedFile == "" {
		logrus.Errorf("Challenge file not found for token: %s", token)
		http.NotFound(w, r)
		return
	}

	data, err := os.ReadFile(path.Join(h.cfg.ChallengePath, matchedFile))
	if err != nil {
		logrus.Errorf("Failed to read challenge file '%s': %s", matchedFile, err)
		http.NotFound(w, r)
		return
	}
	logrus.Infof("Successfully read challenge file: %s", matchedFile)

	w.Header().Set("Content-Type", "text/plain")

	_, err = w.Write(data)
	if err != nil {
		logrus.Errorf("Failed to write challenge response: %s", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	logrus.Infof("Successfully served challenge response for token: %s", token)
}
