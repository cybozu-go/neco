package cmd

import (
	"github.com/spf13/cobra"
)

var createVMXEnabledCommand = &cobra.Command{
	Use:   "vmx-enabled",
	Short: "Create vmx-enabled image",
	Long: `Create vmx-enabled image.

If vmx-enabled image already exists in the project, it is re-created.`,
	Args: cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		return
	},
}

func init() {
	createCmd.AddCommand(createVMXEnabledCommand)
}
