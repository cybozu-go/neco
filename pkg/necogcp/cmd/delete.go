package cmd

import (
	"github.com/spf13/cobra"
)

// deleteCmd is the root subcommand of "necogcp create".
var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "delete related commands",
	Long:  `delete related commands.`,
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
