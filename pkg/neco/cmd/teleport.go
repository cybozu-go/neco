package cmd

import (
	"github.com/spf13/cobra"
)

var teleportCmd = &cobra.Command{
	Use:   "teleport",
	Short: "teleport related commands",
	Long:  `teleport related commands.`,
}

func init() {
	rootCmd.AddCommand(teleportCmd)
}
