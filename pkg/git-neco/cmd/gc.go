package cmd

import (
	"strings"

	"github.com/spf13/cobra"
)

func gcFilterBranch(defBranch string, branches []string) (ret []string) {
	for _, b := range branches {
		switch b {
		case defBranch, "HEAD", "release":
			continue
		}
		if strings.HasPrefix(b, "release-") {
			continue
		}
		ret = append(ret, b)
	}
	return
}

var gcCmd = &cobra.Command{
	Use:   "gc",
	Short: "remove merged remote branches",
	Long:  `Remove merged remote branches from the origin repository.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := sanityCheck(); err != nil {
			return err
		}

		if err := git("fetch", "origin", "--prune"); err != nil {
			return err
		}
		defBranch, err := defaultBranch()
		if err != nil {
			return err
		}
		data, err := gitOutput("branch", "--format=%(refname:lstrip=3)", "-r", "--merged", "origin/"+defBranch)
		if err != nil {
			return err
		}
		for _, b := range gcFilterBranch(defBranch, strings.Fields(string(data))) {
			if err := git("push", "origin", ":"+b); err != nil {
				return err
			}
		}
		if err := git("fetch", "origin", "--prune"); err != nil {
			return err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(gcCmd)
}
