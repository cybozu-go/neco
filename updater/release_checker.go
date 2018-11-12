package updater

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
	"github.com/hashicorp/go-version"
)

// ReleaseChecker checks newer GitHub releases by polling
type ReleaseChecker struct {
	storage   storage.Storage
	github    *ReleaseClient
	leaderKey string

	pkg PackageManager
}

// NewReleaseChecker returns a new ReleaseChecker
func NewReleaseChecker(st storage.Storage, leaderKey string) ReleaseChecker {
	return ReleaseChecker{
		storage:   st,
		pkg:       DebPackageManager{},
		leaderKey: leaderKey,
	}
}

// Run runs newer release during periodic intervals
func (c *ReleaseChecker) Run(ctx context.Context) error {
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

func (c *ReleaseChecker) update(ctx context.Context) error {
	github := c.github
	if c.github == nil {
		var httpc *http.Client
		proxyURL, err := c.storage.GetProxyConfig(ctx)
		if err == storage.ErrNotFound {
		} else if err != nil {
			return err
		} else {
			u, err := url.Parse(proxyURL)
			if err != nil {
				return err
			}
			httpc = newHTTPClient(u)
		}
		github = &ReleaseClient{neco.GitHubRepoOwner, neco.GitHubRepoName, httpc}
	}

	env, err := c.storage.GetEnvConfig(ctx)
	if err != nil {
		return err
	}

	var latest string
	if env == neco.StagingEnv {
		latest, err = github.GetLatestPreReleaseTag(ctx)
	} else if env == neco.ProdEnv {
		latest, err = github.GetLatestReleaseTag(ctx)
	} else {
		log.Warn("Unknown env: "+env, map[string]interface{}{})
		return nil
	}
	if err == ErrNoReleases {
		return nil
	}
	if err != nil {
		return err
	}

	current, err := c.pkg.GetVersion(ctx, neco.NecoPackageName)
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

	return c.storage.UpdateNecoRelease(ctx, latest, c.leaderKey)
}
