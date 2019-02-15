package cmd

import (
	"github.com/spf13/cobra"
)

// necotestCmd is the root subcommand of "necogcp neco-test".
var necotestCmd = &cobra.Command{
	Use:   "neco-test",
	Short: "neco-test related commands",
	Long:  `neco-test related commands.`,
}

func init() {
	rootCmd.AddCommand(necotestCmd)
}
