package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/updater"
	"github.com/cybozu-go/well"
)

func usage() {
	fmt.Fprintln(os.Stderr, "Usage: find-installed-release {staging|production}")
	os.Exit(2)
}

func main() {
	if len(os.Args) != 2 {
		usage()
	}
	datacenter := os.Args[1]

	client := updater.NewReleaseClient(neco.GitHubRepoOwner, neco.GitHubRepoName, http.DefaultClient)

	var getTag func(context.Context) (string, error)
	switch datacenter {
	case "staging":
		getTag = client.GetLatestPublishedTag
	case "production":
		getTag = client.GetLatestReleaseTag
	default:
		usage()
	}

	well.Go(func(ctx context.Context) error {
		tag, err := getTag(ctx)
		if err != nil {
			return err
		}

		fmt.Println(tag)
		return nil
	})
	well.Stop()
	err := well.Wait()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
