package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

var initLocalParams struct {
	name string
}

// initLocalCmd represents the initLocal command
var initLocalCmd = &cobra.Command{
	Use:   "init-local NAME",
	Short: "Initialize data for new application of a boot server executes",
	Long: `Initialize data for new application of a boot server executes. This
command should not be executed more than once.  It asks vault user and
password to generate a vault token, then issue client certificates for
new a application NAME.`,

	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("expected exact one argument")
		}
		initLocalParams.name = args[0]
		return nil
	},

	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("initialize application '%s' in local\n", initLocalParams.name)
	},
}

func init() {
	rootCmd.AddCommand(initLocalCmd)
}
