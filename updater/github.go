package updater

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"strings"

	"github.com/cybozu-go/log"
	"github.com/google/go-github/github"
	version "github.com/hashicorp/go-version"
)

// GetLatestReleaseTag returns latest tag in GitHub Releases of neco repository
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

// GetLatestPreReleaseTag returns latest pre-released tag in GitHub Releases of neco repository
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
	versions := make([]*version.Version, 0, len(releases))
	for i, r := range releases {
		if r.TagName == nil {
			continue
		}
		s := *r.TagName
		trimmed := strings.Split(s, "-")
		// Ignore prefix in tag name.  'prefix-X.Y.Z' is formatted to 'X.Y.Z'
		if len(trimmed) >= 2 {
			s = trimmed[1]
		}
		v, err := version.NewVersion(s)
		if err != nil {
			continue
		}
		versions[i] = v
	}
	sort.Sort(sort.Reverse(version.Collection(versions)))

	if len(versions) == 0 {
		return "", ErrNoReleases
	}

	return versions[0].Original(), nil
}
