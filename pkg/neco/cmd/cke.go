package cmd

import (
	"github.com/spf13/cobra"
)

var ckeCmd = &cobra.Command{
	Use:   "cke",
	Short: "cke related commands",
	Long:  `cke related commands.`,
}

func init() {
	rootCmd.AddCommand(ckeCmd)
}
