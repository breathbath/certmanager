package cmd

import (
	"context"
	"github.com/breathbath/certmanager/pkg/challenge"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"syscall"
)

var challengeCmd = &cobra.Command{
	Use:   "challenge",
	Short: "Starts a challenge server",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

		err := challenge.Start(ctx)
		if err != nil {
			return err
		}

		sig := <-sigs
		logrus.Infof("Received signal %s, shutting down...", sig)

		return nil
	},
}

func initChallengeCmd() {
	RootCmd.AddCommand(challengeCmd)
}
