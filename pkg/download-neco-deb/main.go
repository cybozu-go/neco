package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/updater"
	"github.com/cybozu-go/neco/worker"
	"github.com/cybozu-go/well"
)

func usage() {
	fmt.Fprintln(os.Stderr, "Usage: download-neco-deb {staging|production}")
	os.Exit(2)
}

func main() {
	if len(os.Args) != 2 {
		usage()
	}
	datacenter := os.Args[1]

	ghClient := neco.NewDefaultGitHubClient()
	client := updater.NewReleaseClient(neco.GitHubRepoOwner, neco.GitHubRepoName, ghClient)

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

		deb := &neco.DebianPackage{
			Name:       neco.NecoPackageName,
			Repository: neco.GitHubRepoName,
			Owner:      neco.GitHubRepoOwner,
			Release:    "release-" + tag,
		}
		downloadURL, err := worker.GetGitHubDownloadURL(ctx, ghClient, deb)
		if err != nil {
			return err
		}

		resp, err := http.DefaultClient.Get(downloadURL)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		_, err = io.Copy(os.Stdout, resp.Body)
		if err != nil {
			return err
		}

		return nil
	})
	well.Stop()
	err := well.Wait()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
