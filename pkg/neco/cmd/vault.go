package cmd

import (
	"github.com/spf13/cobra"
)

// vaultCmd is the root subcommand of "neco vault".
var vaultCmd = &cobra.Command{
	Use:   "vault",
	Short: "vault related commands",
	Long:  `vault related commands.`,
}

func init() {
	rootCmd.AddCommand(vaultCmd)
}
