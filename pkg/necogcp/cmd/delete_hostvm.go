package cmd

import (
	"github.com/spf13/cobra"
)

var deleteHostVMCommand = &cobra.Command{
	Use:   "host-vm",
	Short: "Delete host-vm instance",
	Long:  `Delete host-vm instance manually.`,
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		return
	},
}

func init() {
	deleteCmd.AddCommand(deleteHostVMCommand)
}
