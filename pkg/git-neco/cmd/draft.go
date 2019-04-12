package cmd

import (
	"context"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/tcnksm/go-input"
	"regexp"
	"strconv"
)

var draftOpts struct {
	title string
	repo  string
	issue int
}

func taskArguments(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return nil
	}
	if len(args) != 1 {
		return errors.New("too many arguments")
	}

	tmp := args[0]
	reg := regexp.MustCompile(`([a-zA-Z0-9._-]+/[a-zA-Z0-9._-]+)#([0-9]+)$`)
	if m := reg.FindStringSubmatch(args[0]); m != nil {
		draftOpts.repo = m[1]
		tmp = m[2]
	}

	num, err := strconv.Atoi(tmp)
	if err != nil {
		return errors.New("invalid argument")
	}
	draftOpts.issue = num

	return nil
}

var draftCmd = &cobra.Command{
	Use:   "draft [ISSUE]",
	Short: "create a draft pull request for the current branch",
	Long: `Create a draft pull request for the current branch.

If ISSUE is given, this command connect the new pull request with the issue`,

	Args: taskArguments,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDraftCmd(cmd, args, true)
	},
}

func init() {
	draftCmd.Flags().StringVar(&draftOpts.title, "title", "", "title of the pull request")
	rootCmd.AddCommand(draftCmd)
}

func githubClient(ctx context.Context) (GitHubClient, error) {
	token := config.GithubToken
	return NewGitHubClient(ctx, token), nil
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

	ctx := context.Background()
	gc, err := githubClient(ctx)
	if err != nil {
		return err
	}

	curRepoName, curRepoID, err := getCurrentRepo(ctx, gc)
	if err != nil {
		return err
	}

	pr, err := createPR(ctx, gc, curRepoID, branch, title, firstBody, draft)
	if err != nil {
		return err
	}

	if draftOpts.issue != 0 {
		repo := config.GithubRepo
		if draftOpts.repo != "" {
			repo = draftOpts.repo
		}
		issueRepoName, issueRepoID, err := getRepo(ctx, gc, repo)
		if err != nil {
			return err
		}

		fmt.Printf("Connect %s/%s#%d with %s/%s#%d.\n", curRepoName.Owner, curRepoName.Name, pr.Number,
			issueRepoName.Owner, issueRepoName.Name, draftOpts.issue)
		zh := NewZenHubClient(config.ZenhubToken)
		err = zh.Connect(ctx, issueRepoID.Number, draftOpts.issue, curRepoID.Number, pr.Number)
		if err != nil {
			return err
		}
	}

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

func getRepo(ctx context.Context, gc GitHubClient, url string) (*GitHubRepoName, *GitHubRepoID, error) {
	name := ExtractGitHubRepoName(url)
	if name == nil {
		return nil, nil, errors.New("bad URL: " + url)
	}

	id, err := gc.GetRepository(ctx, name)
	if err != nil {
		return nil, nil, err
	}
	return name, id, nil
}

func getCurrentRepo(ctx context.Context, gc GitHubClient) (*GitHubRepoName, *GitHubRepoID, error) {
	origin, err := originURL()
	if err != nil {
		return nil, nil, err
	}
	return getRepo(ctx, gc, origin)
}

// Create a new pull request and add assignee to the pull request.
func createPR(ctx context.Context, gc GitHubClient, repo *GitHubRepoID, branch, title, body string, draft bool) (*PullRequest, error) {
	pr, err := gc.CreatePullRequest(ctx, repo.ID, "master", branch, title, body, draft)
	if err != nil {
		return nil, err
	}
	fmt.Println("\nCreated a pull request:", pr.Permalink)

	assignee, err := gc.GetViewer(ctx)
	if err != nil {
		return nil, err
	}

	err = gc.AddAssigneeToPullRequest(ctx, assignee.ID, pr.ID)
	if err != nil {
		return nil, err
	}
	fmt.Println("Add assignee:", assignee.Login)

	return pr, nil
}
