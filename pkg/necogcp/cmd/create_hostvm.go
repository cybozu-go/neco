package cmd

import (
	"github.com/spf13/cobra"
)

var createHostVMCommand = &cobra.Command{
	Use:   "host-vm",
	Short: "Launch host-vm instance",
	Long: `Launch host-vm instance using vmx-enabled image.

If host-vm instance already exists in the project, it is re-created.`,
	Args: cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		return
	},
}

func init() {
	createCmd.AddCommand(createHostVMCommand)
}
