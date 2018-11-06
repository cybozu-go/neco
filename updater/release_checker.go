package updater

import (
	"context"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
	version "github.com/hashicorp/go-version"
)

// ReleaseChecker is an interface to check new releases
type ReleaseChecker interface {
	Run(ctx context.Context) error
	GetLatest() string
}

// GitHubReleaseChecker checks newer GitHub releases by polling
type GitHubReleaseChecker struct {
	storage storage.Storage
	github  ReleaseInterface

	pkg    PackageManager
	latest atomic.Value
}

// NewReleaseChecker returns a new ReleaseChecker
func NewReleaseChecker(st storage.Storage) ReleaseChecker {
	c := &GitHubReleaseChecker{
		storage: st,
		pkg:     DebPackageManager{},
	}
	c.latest.Store("")
	return c
}

// Run runs newer release at bDaduring periodic intervals
func (c *GitHubReleaseChecker) Run(ctx context.Context) error {
	for {
		interval, err := c.storage.GetCheckUpdateInterval(ctx)
		if err != nil {
			return err
		}

		err = c.update(ctx)
		if err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(interval):
		}
	}
}

// GetLatest returns latest version in GitHub Releases, or returns empty if no
// release are available
func (c *GitHubReleaseChecker) GetLatest() string {
	return c.latest.Load().(string)
}

func (c *GitHubReleaseChecker) update(ctx context.Context) error {
	github := c.github
	if c.github == nil {
		var http *http.Client
		proxyURL, err := c.storage.GetProxyConfig(ctx)
		if err == storage.ErrNotFound {
		} else if err != nil {
			return err
		} else {
			u, err := url.Parse(proxyURL)
			if err != nil {
				return err
			}
			http = neco.NewHTTP(u)
		}
		github = ReleaseClient{neco.GitHubRepoOwner, neco.GitHubRepoName, http}
	}

	env, err := c.storage.GetEnvConfig(ctx)
	if err == storage.ErrNotFound {
		return nil
	} else if err != nil {
		return err
	}

	var latest string
	if env == neco.StagingEnv {
		latest, err = github.GetLatestPreReleaseTag(ctx)
	} else if env == neco.ProdEnv {
		latest, err = github.GetLatestReleaseTag(ctx)
	} else {
		log.Warn("Unknown env: "+env, map[string]interface{}{})
		c.latest.Store("")
		return nil
	}
	if err == ErrNoReleases {
		return nil
	}
	if err != nil {
		return err
	}

	current, err := c.pkg.GetVersion(ctx, "neco")
	if err != nil {
		return err
	}

	latestVer, err := version.NewVersion(latest)
	if err != nil {
		return err
	}

	currentVer, err := version.NewVersion(current)
	if err != nil {
		return err
	}

	if !latestVer.GreaterThan(currentVer) {
		return nil
	}

	log.Info("New neco release is found ", map[string]interface{}{
		"env":     env,
		"version": latest,
	})
	c.latest.Store(latest)
	return nil
}
