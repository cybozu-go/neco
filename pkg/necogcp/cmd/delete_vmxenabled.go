package cmd

import (
	"github.com/spf13/cobra"
)

var deleteVMXEnabledCommand = &cobra.Command{
	Use:   "vmx-enabled",
	Short: "Delete vmx-enabled image",
	Long:  `Delete vmx-enabled image.`,
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		return
	},
}

func init() {
	deleteCmd.AddCommand(deleteVMXEnabledCommand)
}
