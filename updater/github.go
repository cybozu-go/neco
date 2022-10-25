package updater

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"strings"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/google/go-github/v48/github"
	version "github.com/hashicorp/go-version"
)

func trimTagName(s string) string {
	trimmed := strings.SplitN(s, "-", 2)
	// Ignore prefix in tag name.  'prefix-X.Y.Z' is formatted to 'X.Y.Z'
	if len(trimmed) >= 2 {
		return trimmed[1]
	}
	return s
}

// ReleaseClient gets GitHub Releases
type ReleaseClient struct {
	owner string
	repo  string
	http  *http.Client
}

// NewReleaseClient returns ReleaseClient
func NewReleaseClient(owner, repo string, http *http.Client) *ReleaseClient {
	return &ReleaseClient{owner, repo, http}
}

// GetLatestReleaseTag returns latest published release tag in GitHub Releases of neco repository
func (c ReleaseClient) GetLatestReleaseTag(ctx context.Context) (string, error) {
	client := neco.NewGitHubClient(c.http)
	release, _, err := client.Repositories.GetLatestRelease(ctx, c.owner, c.repo)
	if err != nil {
		log.Warn("failed to get the latest GitHub release", map[string]interface{}{
			"owner":      c.owner,
			"repository": c.repo,
			log.FnError:  err,
		})
		return "", ErrNoReleases
	}
	if release.TagName == nil {
		log.Error("no tagged release", map[string]interface{}{
			"owner":      c.owner,
			"repository": c.repo,
			"release":    release.String(),
		})
		return "", errors.New("no tagged release")
	}
	return trimTagName(*release.TagName), nil
}

// GetLatestPublishedTag returns latest published release/pre-release tag in GitHub Releases of neco repository
func (c ReleaseClient) GetLatestPublishedTag(ctx context.Context) (string, error) {
	client := neco.NewGitHubClient(c.http)

	opt := &github.ListOptions{
		PerPage: 100,
	}

	var releases []*github.RepositoryRelease
	for {
		rs, resp, err := client.Repositories.ListReleases(ctx, c.owner, c.repo, opt)
		if err != nil {
			log.Warn("failed to list GitHub releases", map[string]interface{}{
				"owner":      c.owner,
				"repository": c.repo,
				log.FnError:  err,
			})
			return "", ErrNoReleases
		}
		releases = append(releases, rs...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	versions := make([]*version.Version, 0, len(releases))
	for _, r := range releases {
		if r.TagName == nil || r.GetDraft() {
			continue
		}
		s := trimTagName(*r.TagName)
		v, err := version.NewVersion(s)
		if err != nil {
			continue
		}
		versions = append(versions, v)
	}
	sort.Sort(sort.Reverse(version.Collection(versions)))

	if len(versions) == 0 {
		return "", ErrNoReleases
	}

	return versions[0].Original(), nil
}
