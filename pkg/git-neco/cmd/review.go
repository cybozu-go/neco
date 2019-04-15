package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var reviewCmd = &cobra.Command{
	Use:   "review [ISSUE]",
	Short: "mark draft PR ready for review, or create a one",
	Long: `Mark a draft PR ready for review if such a PR exists
for the current branch.  If no such PR exists, this command works
just like "draft" but do not make the PR as draft.

If draft PR is not found and ISSUE is given, this command connects
the new pull request with the issue.
The ISSUE can be specified in one of the following formats.
  - <issue number>
  - <owner>/<repo>#<issue number>
  - https://github.com/<owner>/<repo>#<issue number>
  - git@github.com:<owner>/<repo>#<issue number>`,
	Args: taskArguments,
	RunE: runReviewCmd,
}

func init() {
	reviewCmd.Flags().StringVar(&draftOpts.title, "title", "", "title of the pull request")
	rootCmd.AddCommand(reviewCmd)
}

func runReviewCmd(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	gc, err := githubClient(ctx)
	if err != nil {
		return err
	}

	repo, err := getCurrentRepo(ctx, gc)
	if err != nil {
		return err
	}

	br, err := currentBranch()
	if err != nil {
		return err
	}

	pr, err := gc.GetDraftPR(ctx, repo, br)
	if err != nil {
		return err
	}

	if pr == "" {
		fmt.Println("Draft pull request is not found.  Creating a new pull request...")
		return runDraftCmd(cmd, args, false)
	}

	err = gc.MarkDraftReadyForReview(ctx, pr)
	if err != nil {
		return err
	}

	fmt.Println("Marked draft pull request ready for review")
	return nil
}
