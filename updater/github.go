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

// ReleaseInterface is an interface to fetch latest release and pre-release
type ReleaseInterface interface {
	GetLatestReleaseTag(ctx context.Context) (string, error)
	GetLatestPreReleaseTag(ctx context.Context) (string, error)
}

// ReleaseClient is an implementation of ReleaseInterface to get GitHub Releases
type ReleaseClient struct {
	owner string
	repo  string
	http  *http.Client
}

// GetLatestReleaseTag returns latest tag in GitHub Releases of neco repository
func (c ReleaseClient) GetLatestReleaseTag(ctx context.Context) (string, error) {
	client := github.NewClient(c.http)
	release, resp, err := client.Repositories.GetLatestRelease(ctx, c.owner, c.repo)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			return "", ErrNoReleases
		}
		log.Error("failed to get the latest GitHub release", map[string]interface{}{
			"owner":      c.owner,
			"repository": c.repo,
			log.FnError:  err,
		})
		return "", err
	}
	if release.TagName == nil {
		log.Error("no tagged release", map[string]interface{}{
			"owner":      c.owner,
			"repository": c.repo,
			"release":    release.String(),
		})
		return "", errors.New("no tagged release")
	}
	return *release.TagName, nil
}

// GetLatestPreReleaseTag returns latest pre-released tag in GitHub Releases of neco repository
func (c ReleaseClient) GetLatestPreReleaseTag(ctx context.Context) (string, error) {
	client := github.NewClient(c.http)

	opt := &github.ListOptions{
		PerPage: 100,
	}

	var releases []*github.RepositoryRelease
	for {
		rs, resp, err := client.Repositories.ListReleases(ctx, c.owner, c.repo, opt)
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
	for _, r := range releases {
		if r.TagName == nil || !r.GetPrerelease() {
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
		versions = append(versions, v)
	}
	sort.Sort(sort.Reverse(version.Collection(versions)))

	if len(versions) == 0 {
		return "", ErrNoReleases
	}

	return versions[0].Original(), nil
}
