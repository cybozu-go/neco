package cmd

import (
	"github.com/spf13/cobra"
)

var serfTagCmd = &cobra.Command{
	Use:   "serf-tag",
	Short: "serf-tag related commands",
	Long:  `serf-tag related commands.`,
}

func init() {
	rootCmd.AddCommand(serfTagCmd)
}
