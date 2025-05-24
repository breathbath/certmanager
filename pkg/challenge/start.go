package challenge

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"net/http"
	"time"
)

func Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/acme-challenge/", challengeHandler)

	cfg, err := LoadConfig()
	if err != nil {
		return err
	}

	addr := fmt.Sprintf(":%d", cfg.Port)

	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Run server in a goroutine
	go func() {
		logrus.Infof("Starting HTTP server on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("HTTP server error: %v", err)
		}
	}()

	<-ctx.Done()
	logrus.Info("Shutting down HTTP server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = srv.Shutdown(shutdownCtx)
	if err != nil {
		return err
	}

	return nil
}
