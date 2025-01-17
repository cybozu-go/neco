package updater

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
	"github.com/google/go-github/v50/github"
	"github.com/robfig/cron/v3"
)

var (
	defaultCheckTimes    = "* * * * *"
	defaultCheckTimeZone = "Asia/Tokyo"
)

// ReleaseChecker checks newer GitHub releases by polling
type ReleaseChecker struct {
	storage   storage.Storage
	leaderKey string
	ghClient  *github.Client

	check         func(context.Context) (string, error)
	checkTimes    []cron.Schedule
	checkTimeZone time.Location
	current       string
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
	rt, err := c.storage.GetReleaseTime(ctx)
	if err != nil {
		if err == storage.ErrNotFound {
			rt = defaultCheckTimes
		} else {
			return err
		}
	}
	rts := strings.Split(rt, ",")
	for _, rt := range rts {
		checkTime, err := cron.ParseStandard(rt)
		if err != nil {
			return err
		}
		c.checkTimes = append(c.checkTimes, checkTime)
	}

	tz, err := c.storage.GetReleaseTimeZone(ctx)
	if err != nil {
		if err == storage.ErrNotFound {
			tz = defaultCheckTimeZone
		} else {
			return err
		}
	}
	location, err := time.LoadLocation(tz)
	if err != nil {
		return err
	}
	c.checkTimeZone = *location

	switch env {
	case neco.NoneEnv:
		c.check = func(ctx context.Context) (string, error) {
			return "", ErrNoReleases
		}
	case neco.TestEnv:
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
	now := time.Now().In(&c.checkTimeZone)
	isWithinSchedule := false
	for _, schedule := range c.checkTimes {
		next := schedule.Next(now)
		// If the next scheduled time is within 1 minutes from now, we consider that it's within the schedule.
		if next.Sub(now) <= time.Second*60 {
			isWithinSchedule = true
			break
		} else {
			continue
		}
	}
	if !isWithinSchedule {
		log.Info("not within schedule", map[string]interface{}{
			"now":       now,
			"schedules": c.checkTimes,
		})
		return nil
	}

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
