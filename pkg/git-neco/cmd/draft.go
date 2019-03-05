package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"

	"github.com/cybozu-go/log"
	kintone "github.com/kintone/go-kintone"
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
	Use:   "draft [TASK]",
	Short: "create a draft pull request for the current branch",
	Long: `Create a draft pull request for the current branch.

If TASK is given, this command edits kintone record for TASK to
add URL of the new pull request.`,

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

	var app *kintone.App
	if draftOpts.recordID != 0 {
		app, err = newAppClient(config.KintoneURL, config.KintoneToken)
	}

	if ok, err := confirmTask(app, draftOpts.recordID); err != nil {
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

	if err := addURLToRecord(app, draftOpts.recordID, prURL); err != nil {
		return err
	}
	if app != nil {
		fmt.Println("\nUpdated kintone record.")
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

func confirmTask(app *kintone.App, id uint64) (bool, error) {
	if app == nil {
		return true, nil
	}

	rec, err := app.GetRecord(id)
	if err != nil {
		return false, err
	}

	field := rec.Fields["summary"]
	summary, ok := field.(kintone.SingleLineTextField)
	if !ok {
		return false, fmt.Errorf("unexpected summary field: %T, %v", field, field)
	}

	fmt.Printf("NecoTask-%d: %s\n", id, string(summary))
	return askYorN("Is this ok?")
}

func githubClientForRepo(ctx context.Context, repo GitHubRepo) (GitHubClient, error) {
	endpoint, _ := url.Parse("https://api.github.com/graphql")
	token := config.GithubToken

	switch repo.Owner {
	case "Neco":
		u, err := url.Parse(config.GheURL)
		if err != nil {
			return GitHubClient{}, err
		}
		u.Path = "/api/graphql"
		endpoint = u
		token = config.GheToken
	}
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

func addURLToRecord(app *kintone.App, id uint64, prURL string) error {
	if app == nil {
		return nil
	}

	rec, err := app.GetRecord(id)
	if err != nil {
		return err
	}
	field, ok := rec.Fields["prs"]
	if !ok {
		log.Warn("development detail field is not found in kintone app", nil)
		return nil
	}
	dd, ok := field.(kintone.MultiLineTextField)
	if !ok {
		return fmt.Errorf("unexpected field type: %T, %v", field, field)
	}
	if dd == "" {
		dd = kintone.MultiLineTextField(prURL)
	} else {
		dd = kintone.MultiLineTextField(fmt.Sprintf("%s\n%s", string(dd), prURL))
	}
	rec.Fields = map[string]interface{}{"prs": dd}

	return app.UpdateRecord(rec, true)
}
