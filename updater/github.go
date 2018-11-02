package updater

import (
	"context"
	"errors"
	"net/http"

	"github.com/cybozu-go/log"
	"github.com/google/go-github/github"
)

func GetLatestReleaseTag(ctx context.Context) (string, error) {
	client := github.NewClient(nil)
	release, resp, err := client.Repositories.GetLatestRelease(ctx, "cybozu-go", "neco")
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			return "", ErrNoReleases
		}
		log.Error("failed to get the latest GitHub release", map[string]interface{}{
			"owner":      "cybozu-go",
			"repository": "neco",
			log.FnError:  err,
		})
		return "", err
	}
	if release.TagName == nil {
		log.Error("no tagged release", map[string]interface{}{
			"owner":      "cybozu-go",
			"repository": "neco",
			"release":    release.String(),
		})
		return "", errors.New("no tagged release")
	}
	return *release.TagName, nil
}

func GetLatestPreReleaseTag(ctx context.Context) (string, error) {
	client := github.NewClient(nil)

	opt := &github.ListOptions{
		PerPage: 100,
	}

	var releases []*github.RepositoryRelease
	for {
		rs, resp, err := client.Repositories.ListReleases(ctx, "cybozu-go", "neco", opt)
		if err != nil {
			return "", err
		}
		releases = append(releases, rs...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	var release *github.RepositoryRelease
	for _, r := range releases {
		if r.Prerelease != nil && *r.Prerelease {
			release = r
			break
		}
	}
	if release == nil {
		return "", ErrNoReleases
	}
	if release.TagName == nil {
		log.Error("no tagged release", map[string]interface{}{
			"owner":      "cybozu-go",
			"repository": "neco",
			"release":    release.String(),
		})
		return "", errors.New("no tagged release")
	}
	return *release.TagName, nil
}
