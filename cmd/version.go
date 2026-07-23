package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version is the built-in version of the hydra CLI. It can be overridden at
// build time with -ldflags "-X github.com/hydradns/hydra-cli/cmd.Version=vX.Y.Z".
var Version = "1.0.0"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the hydra CLI version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("hydra %s\n", Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
