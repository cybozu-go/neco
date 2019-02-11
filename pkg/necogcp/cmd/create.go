package cmd

import (
	"github.com/spf13/cobra"
)

// createCmd is the root subcommand of "necogcp create".
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "create related commands",
	Long:  `create related commands.`,
}

func init() {
	rootCmd.AddCommand(createCmd)
}
