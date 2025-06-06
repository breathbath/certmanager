package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of winebot",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Winebot Version:\n%s", Version)
	},
}

func initVersionCmd() {
	RootCmd.AddCommand(versionCmd)
}
