package cmd

import (
	"github.com/spf13/cobra"
)

// configCmd is the root subcommand of "neco config".
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "config related commands",
	Long:  `config related commands.`,
}

func init() {
	rootCmd.AddCommand(configCmd)
}
