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
	prefix := "/.well-known/acme-challenge/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		logrus.Errorf("Invalid request path: %s", r.URL.Path)

		http.NotFound(w, r)
		return
	}

	token := strings.TrimPrefix(r.URL.Path, prefix)
	if token == "" {
		logrus.Errorf("Invalid request path: %s", r.URL.Path)
		http.NotFound(w, r)
		return
	}

	files, err := os.ReadDir(h.cfg.ChallengePath)
	if err != nil {
		logrus.Errorf("Failed to read challenge directory: %s", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var matchedFile string
	for _, f := range files {
		if !f.IsDir() && f.Name() == token {
			matchedFile = f.Name()
			break
		}
	}
	if matchedFile == "" {
		logrus.Errorf("Challenge file not found: %s", token)
		http.NotFound(w, r)
		return
	}

	data, err := os.ReadFile(path.Join(h.cfg.ChallengePath, matchedFile))
	if err != nil {
		logrus.Errorf("Failed to read challenge file: %s", err)
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/plain")

	_, err = w.Write(data)
	if err != nil {
		logrus.Errorf("Failed to write challenge response: %s", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
