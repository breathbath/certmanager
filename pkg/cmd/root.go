package cmd

import (
	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "certmanager",
	Short: "Root Command for certmanager",
}

func Execute() error {
	initCertManagerCmd()
	initChallengeCmd()
	initVersionCmd()
	return RootCmd.Execute()
}
