package cmd

import (
	"github.com/spf13/cobra"
)

var bmcCmd = &cobra.Command{
	Use:   "bmc",
	Short: "bmc related commands",
	Long:  `bmc related commands.`,
}

func init() {
	rootCmd.AddCommand(bmcCmd)
}
