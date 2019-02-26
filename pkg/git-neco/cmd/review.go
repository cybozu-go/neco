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

var reviewOpts struct {
	title    string
	recordID uint64
}

var reviewCmd = &cobra.Command{
	Use:     "review [TASK]",
	Aliases: []string{"draft"},
	Short:   "create pull request for the current branch",
	Long: `Create a new pull request for the current branch.

If TASK is given, this command edits kintone record for TASK to
add URL of the new pull request.

When called as "draft", the pull request will be created as draft.`,

	Args: func(cmd *cobra.Command, args []string) error {
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
		reviewOpts.recordID = id
		return nil
	},

	RunE: func(cmd *cobra.Command, args []string) error {
		draft := cmd.CalledAs() == "draft"

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
		if reviewOpts.recordID != 0 {
			app, err = newAppClient(config.KintoneURL, config.KintoneToken)
		}

		if ok, err := confirmTask(app, reviewOpts.recordID); err != nil {
			return err
		} else if !ok {
			return nil
		}

		if err := git("push", "-u", "origin", branch+":"+branch); err != nil {
			return err
		}
		fmt.Printf("git push to origin/%s succeeded\n", branch)

		title := reviewOpts.title
		if title == "" {
			title = firstSummary
		}

		prURL, err := createPR(branch, title, firstBody, draft)
		if err != nil {
			return err
		}
		fmt.Println("Created a pull request:", prURL)

		return addURLToRecord(app, reviewOpts.recordID, prURL)
	},
}

func init() {
	reviewCmd.Flags().StringVar(&reviewOpts.title, "title", "", "title of the pull request")
	rootCmd.AddCommand(reviewCmd)
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

func createPR(branch, title, body string, draft bool) (string, error) {
	repo, err := repoName()
	if err != nil {
		return "", err
	}

	endpoint, _ := url.Parse("https://api.github.com/graphql")
	token := config.GithubToken
	switch repo {
	case "wiki":
		u, err := url.Parse(config.GheURL)
		if err != nil {
			return "", err
		}
		u.Path = "/api/graphql"
		endpoint = u
		token = config.GheToken
	}

	ctx := context.Background()
	gc := NewGitHubClient(ctx, endpoint, token)
	return gc.CreatePullRequest(ctx, "master", branch, title, body, draft)
}

func addURLToRecord(app *kintone.App, id uint64, prURL string) error {
	rec, err := app.GetRecord(id)
	if err != nil {
		return err
	}
	field, ok := rec.Fields["Development_Detail"]
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
	rec.Fields["Development_Detail"] = dd

	return app.UpdateRecord(rec, true)
}
