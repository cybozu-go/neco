package cmd

import (
	"strings"

	"github.com/spf13/cobra"
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "remove local branches merged into default branch",
	Long:  `Remove local branches merged into default branch.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := sanityCheck(); err != nil {
			return err
		}

		defBranch, err := defaultBranch()
		if err != nil {
			return err
		}
		if err := git("checkout", defBranch); err != nil {
			return err
		}
		if err := git("fetch", "origin"); err != nil {
			return err
		}
		data, err := gitOutput("branch", "--format=%(refname:short)", "--merged", "origin/"+defBranch)
		if err != nil {
			return err
		}
		for _, b := range gcFilterBranch(defBranch, strings.Fields(string(data))) {
			if err := git("branch", "-D", b); err != nil {
				return err
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(cleanCmd)
}
