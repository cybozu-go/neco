package updater

import (
	"context"
	"errors"
	"strings"

	"github.com/cybozu-go/log"
	"github.com/google/go-github/v50/github"
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
	owner     string
	repo      string
	tagPrefix string
	ghClient  *github.Client
}

// NewReleaseClient returns ReleaseClient
func NewReleaseClient(owner, repo string, ghClient *github.Client) *ReleaseClient {
	return &ReleaseClient{owner, repo, "release-", ghClient}
}

func (c *ReleaseClient) SetTagPrefix(tagPrefix string) {
	c.tagPrefix = tagPrefix
}

// GetLatestReleaseTag returns latest published release tag in GitHub Releases of neco repository
// In this function, "latest" means the release which is marked "latest" on GitHub.
func (c *ReleaseClient) GetLatestReleaseTag(ctx context.Context) (string, error) {
	release, _, err := c.ghClient.Repositories.GetLatestRelease(ctx, c.owner, c.repo)
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
// In this Function, "latest" means the release whose version part is the greatest.
func (c *ReleaseClient) GetLatestPublishedTag(ctx context.Context) (string, error) {
	opt := &github.ListOptions{
		PerPage: 100,
	}

	var latest *version.Version

	for {
		releases, resp, err := c.ghClient.Repositories.ListReleases(ctx, c.owner, c.repo, opt)
		if err != nil {
			log.Warn("failed to list GitHub releases", map[string]interface{}{
				"owner":      c.owner,
				"repository": c.repo,
				log.FnError:  err,
			})
			return "", ErrNoReleases
		}

		for _, r := range releases {
			if r.TagName == nil || r.GetDraft() {
				continue
			}
			if !strings.HasPrefix(*r.TagName, c.tagPrefix) {
				continue
			}
			s := trimTagName(*r.TagName)
			v, err := version.NewVersion(s)
			if err != nil {
				continue
			}
			if latest == nil || v.Compare(latest) > 0 {
				latest = v
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	if latest == nil {
		return "", ErrNoReleases
	}

	return latest.Original(), nil
}
