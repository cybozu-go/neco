package cmd

import (
	"github.com/spf13/cobra"
)

var sessionLogCmd = &cobra.Command{
	Use:   "session-log",
	Short: "session log related commands",
	Long:  `session log related commands.`,
}

func init() {
	rootCmd.AddCommand(sessionLogCmd)
}
