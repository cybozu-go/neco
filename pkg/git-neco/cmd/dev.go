package cmd

import (
	"github.com/spf13/cobra"
)

var devCmd = &cobra.Command{
	Use:   "dev BRANCH_NAME",
	Short: "create a feature branch from the latest default branch",
	Long:  `Create a feature branch from the latest default branch.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := sanityCheck(); err != nil {
			return err
		}

		if err := git("fetch", "origin"); err != nil {
			return err
		}

		defBranch, err := defaultBranch()
		if err != nil {
			return err
		}
		return git("checkout", "--no-track", "-b", args[0], "origin/"+defBranch)
	},
}

func init() {
	rootCmd.AddCommand(devCmd)
}
