package updater

import (
	"context"
	"errors"
	"reflect"
	"time"

	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
	version "github.com/hashicorp/go-version"
)

// Action is the type for NextAction.
type Action int

// Actions.
const (
	ActionError Action = iota
	ActionWaitInfo
	ActionReconfigure
	ActionNewVersion
	ActionWaitWorkers
	ActionStop
	ActionWaitClear
)

func (a Action) String() string {
	switch a {
	case ActionError:
		return "error"
	case ActionWaitInfo:
		return "wait-for-info"
	case ActionReconfigure:
		return "reconfigure"
	case ActionNewVersion:
		return "request-update"
	case ActionWaitWorkers:
		return "wait-for-workers"
	case ActionStop:
		return "request-stop"
	case ActionWaitClear:
		return "wait-for-user-recovery"
	default:
		panic("no such action")
	}
}

// NextAction decides the next action to do for neco-updater.
func NextAction(ctx context.Context, ss *storage.Snapshot, pkg PackageManager, timeout time.Duration) (Action, error) {
	if ss.Latest == "" {
		return ActionWaitInfo, nil
	}
	latestVer, err := version.NewVersion(ss.Latest)
	if err != nil {
		return ActionError, err
	}

	current, err := pkg.GetVersion(ctx, neco.NecoPackageName)
	if err != nil {
		return ActionError, err
	}
	if current == "" {
		return ActionError, errors.New("neco package is not installed")
	}
	currentVer, err := version.NewVersion(current)
	if err != nil {
		return ActionError, err
	}
	if ss.Request == nil {
		if latestVer.GreaterThan(currentVer) {
			return ActionNewVersion, nil
		}
		return ActionWaitInfo, nil
	}

	if ss.Request.Stop {
		return ActionWaitClear, nil
	}

	// in case some workers have aborted but request is not yet stopped,
	// the request need to be stopped.
	for _, status := range ss.Statuses {
		if status.Version != ss.Request.Version {
			continue
		}
		if status.Cond == neco.CondAbort {
			return ActionStop, nil
		}
	}

	if !neco.UpdateCompleted(ss.Request.Version, ss.Request.Servers, ss.Statuses) {
		if time.Now().Sub(ss.Request.StartedAt) > timeout {
			return ActionStop, nil
		}
		return ActionWaitWorkers, nil
	}

	// reconfigure the new set of boot servers with unchanged neco package version.
	if !reflect.DeepEqual(ss.Request.Servers, ss.Servers) {
		return ActionReconfigure, nil
	}

	requestVer, err := version.NewVersion(ss.Request.Version)
	if err != nil {
		return ActionError, err
	}
	if !latestVer.Equal(requestVer) {
		return ActionNewVersion, nil
	}

	return ActionWaitInfo, nil
}
