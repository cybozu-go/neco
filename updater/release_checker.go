package updater

import (
	"context"
	"errors"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
	"github.com/google/go-github/v50/github"
)

// ReleaseChecker checks newer GitHub releases by polling
type ReleaseChecker struct {
	storage   storage.Storage
	leaderKey string
	ghClient  *github.Client

	check   func(context.Context) (string, error)
	current string
}

// NewReleaseChecker returns a new ReleaseChecker
func NewReleaseChecker(st storage.Storage, leaderKey string, ghClient *github.Client) ReleaseChecker {
	return ReleaseChecker{
		storage:   st,
		leaderKey: leaderKey,
		ghClient:  ghClient,
	}
}

// Run periodically checks the new release of neco package at GitHub.
func (c *ReleaseChecker) Run(ctx context.Context) error {
	github := NewReleaseClient(neco.GitHubRepoOwner, neco.GitHubRepoName, c.ghClient)

	env, err := c.storage.GetEnvConfig(ctx)
	if err != nil {
		return err
	}

	switch env {
	case neco.NoneEnv:
		c.check = func(ctx context.Context) (string, error) {
			return "", ErrNoReleases
		}
	case neco.TestEnv:
		log.Info("running in test environment", nil)
		c.check = func(ctx context.Context) (string, error) {
			return "9999.12.31-99999", nil
		}
	case neco.DevEnv:
		github.SetTagPrefix("test-")
		c.check = github.GetLatestPublishedTag
	case neco.StagingEnv:
		c.check = github.GetLatestPublishedTag
	case neco.ProdEnv:
		c.check = github.GetLatestReleaseTag
	default:
		return errors.New("unknown env: " + env)
	}

	current, err := c.storage.GetNecoRelease(ctx)
	if err != nil {
		return err
	}
	c.current = current

	interval, err := c.storage.GetCheckUpdateInterval(ctx)
	if err != nil {
		return err
	}

	for {
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
	latest, err := c.check(ctx)
	if err == ErrNoReleases {
		return nil
	}
	if err != nil {
		return err
	}

	if latest == c.current {
		return nil
	}

	currentRelease, err := newNecoRelease(c.current)
	if err != nil {
		return err
	}
	latestRelease, err := newNecoRelease(latest)
	if err != nil {
		return err
	}

	isNewer := latestRelease.isNewerThan(currentRelease)
	if !isNewer {
		log.Info("got neco release with older version", map[string]interface{}{
			"latest":  latest,
			"current": c.current,
		})
		return nil
	}

	c.current = latest
	log.Info("found a new neco release", map[string]interface{}{
		"version": latest,
	})

	return c.storage.UpdateNecoRelease(ctx, latest, c.leaderKey)
}
