package cmd

import (
	"context"
	"github.com/breathbath/certmanager/pkg/certmanager"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"syscall"
)

var certManagerCmd = &cobra.Command{
	Use:   "certmanager",
	Short: "Starts a certmanager which checks and updates certificates if needed",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

		cm, err := certmanager.NewCertManager()
		if err != nil {
		}

		err = cm.Start(ctx)
		if err != nil {
		}

		sig := <-sigs
		logrus.Infof("Received signal %s, shutting down...", sig)

		return nil
	},
}

func initCertManagerCmd() {
	RootCmd.AddCommand(certManagerCmd)
}
