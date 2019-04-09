package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"

	"github.com/spf13/cobra"
	input "github.com/tcnksm/go-input"
)

var draftOpts struct {
	title    string
	recordID uint64
}

func taskArguments(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return nil
	}
	if len(args) != 1 {
		return errors.New("too many arguments")
	}
	id, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return err
	}
	draftOpts.recordID = id
	return nil
}

var draftCmd = &cobra.Command{
	Use:   "draft [ISSUE]",
	Short: "create a draft pull request for the current branch",
	Long:  `Create a draft pull request for the current branch.`,

	Args: taskArguments,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDraftCmd(cmd, args, true)
	},
}

func init() {
	draftCmd.Flags().StringVar(&draftOpts.title, "title", "", "title of the pull request")
	rootCmd.AddCommand(draftCmd)
}

func runDraftCmd(cmd *cobra.Command, args []string, draft bool) error {
	branch, err := currentBranch()
	if err != nil {
		return err
	}
	if branch == "master" {
		return errors.New("direct push to master is prohibited")
	}

	_, firstSummary, firstBody, err := firstUnmerged()
	if err != nil {
		return err
	}

	if ok, err := confirmUncommitted(); err != nil {
		return err
	} else if !ok {
		return nil
	}

	if err := git("push", "-u", "origin", branch+":"+branch); err != nil {
		return err
	}

	title := draftOpts.title
	if title == "" {
		title = firstSummary
	}

	prURL, err := createPR(branch, title, firstBody, draft)
	if err != nil {
		return err
	}
	fmt.Println("\nCreated a pull request:", prURL)

	return nil
}

func askYorN(query string) (bool, error) {
	ans, err := input.DefaultUI().Ask(query+" [y/N]", &input.Options{
		Default:     "N",
		HideDefault: true,
		HideOrder:   true,
	})
	if err != nil {
		return false, err
	}
	switch ans {
	case "y", "Y", "yes", "YES":
		return true, nil
	}
	return false, nil
}

func confirmUncommitted() (bool, error) {
	ok, err := checkUncommittedFiles()
	if err != nil {
		return false, err
	}
	if ok {
		return true, nil
	}

	fmt.Println("WARNING: you have uncommitted files.")
	return askYorN("Continue?")
}

func githubClientForRepo(ctx context.Context, repo GitHubRepo) (GitHubClient, error) {
	endpoint, _ := url.Parse("https://api.github.com/graphql")
	token := config.GithubToken
	return NewGitHubClient(ctx, endpoint, token), nil
}

func createPR(branch, title, body string, draft bool) (string, error) {
	repo, err := CurrentRepo()
	if err != nil {
		return "", err
	}

	ctx := context.Background()
	gc, err := githubClientForRepo(ctx, *repo)
	if err != nil {
		return "", err
	}

	repoID, err := gc.Repository(ctx, *repo)
	if err != nil {
		return "", err
	}
	return gc.CreatePullRequest(ctx, repoID, "master", branch, title, body, draft)
}
