package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var reviewOpts struct {
	title    string
	recordID uint64
}

var reviewCmd = &cobra.Command{
	Use:   "review [TASK]",
	Short: "create a pull request for the current branch",
	Long: `Create a new pull request for the current branch.

If TASK is given, this command edits kintone record for TASK to
add URL of the new pull request.`,

	Args: taskArguments,
	RunE: runReviewCmd,
}

func init() {
	reviewCmd.Flags().StringVar(&reviewOpts.title, "title", "", "title of the pull request")
	rootCmd.AddCommand(reviewCmd)
}

func runReviewCmd(cmd *cobra.Command, args []string) error {
	endpoint, token, err := GetEndpointInfo()
	if err != nil {
		return err
	}
	ctx := context.Background()
	gc := NewGitHubClient(ctx, endpoint, token)
	rep, err := CurrentRepo()
	if err != nil {
		return err
	}

	br, err := currentBranch()
	if err != nil {
		return err
	}

RETRY:
	id, err := gc.GetDraftPullRequest(ctx, rep.Owner, rep.Name, br)
	if err != nil {
		return err
	}
	if id == "" {
		fmt.Println("Draft pull request is not found.  Creating a new draft pull request...")
		err := runDraftCmd(cmd, args)
		if err != nil {
			return err
		}
		goto RETRY
	}

	err = gc.MarkReadyForReview(ctx, id)
	if err != nil {
		return err
	}
	fmt.Println("Marked draft pull request ready for review")
	return nil
}
