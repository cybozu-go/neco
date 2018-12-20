package cmd

import (
	"github.com/spf13/cobra"
)

// sshCmd is the root subcommand of "neco ssh".
var sshCmd = &cobra.Command{
	Use:   "ssh",
	Short: "ssh related commands",
	Long:  `ssh related commands.`,
}

func init() {
	rootCmd.AddCommand(sshCmd)
}
