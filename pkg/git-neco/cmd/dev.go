package cmd

import (
	"github.com/spf13/cobra"
)

var devCmd = &cobra.Command{
	Use:   "dev BRANCH_NAME",
	Short: "create a feature branch from the latest origin/master",
	Long:  `Create a feature branch from the latest origin/master.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := sanityCheck(); err != nil {
			return err
		}

		if err := git("fetch", "origin"); err != nil {
			return err
		}

		return git("checkout", "--no-track", "-b", args[0], "origin/master")
	},
}

func init() {
	rootCmd.AddCommand(devCmd)
}
