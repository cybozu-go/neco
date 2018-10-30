package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

var initParams struct {
	name string
}

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init NAME",
	Short: "Initialize data for new application of the cluster",
	Long: `Initialize data for new application of the cluster.
Setup etcd user/role for a new application NAME. This command should not be
executed more than once.`,

	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("expected exact one argument")
		}
		initParams.name = args[0]
		return nil
	},

	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("initialize application '%s'\n", initParams.name)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
