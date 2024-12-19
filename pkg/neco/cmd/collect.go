package cmd

import (
	"github.com/spf13/cobra"
)

var collectCmd = &cobra.Command{
	Use:   "collect",
	Short: "collect related commands",
	Long:  `collect`,
}

func init() {
	rootCmd.AddCommand(collectCmd)
}
