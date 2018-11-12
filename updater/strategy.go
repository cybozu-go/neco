package updater

import (
	"context"
	"reflect"

	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
	version "github.com/hashicorp/go-version"
)

type Action int

const (
	ActionNone Action = iota
	ActionNewVersion
	ActionWaitClear
	ActionReconfigure
	ActionNotifyCompleted
)

func (s Server) NextAction(ctx context.Context, ss *storage.Snapshot, pkg PackageManager) (Action, error) {
	latestVer, err := version.NewVersion(ss.Latest)
	if err != nil {
		return ActionNone, err
	}

	current, err := pkg.GetVersion(ctx, neco.NecoPackageName)
	if err != nil {
		return ActionNone, err
	}
	if current == "" {
		return ActionNewVersion, nil
	}
	currentVer, err := version.NewVersion(current)
	if err != nil {
		return ActionNone, err
	}
	if ss.Request == nil {
		if latestVer.GreaterThan(currentVer) {
			return ActionNewVersion, nil
		}
		return ActionNone, nil
	}

	if ss.Request.Stop {
		return ActionWaitClear, nil
	}

	if !reflect.DeepEqual(ss.Request.Servers, ss.Servers) {
		return ActionReconfigure, nil
	}

	requestVer, err := version.NewVersion(ss.Request.Version)
	if err != nil {
		return ActionNone, err
	}
	if !latestVer.Equal(requestVer) {
		return ActionNewVersion, nil
	}

	if neco.UpdateCompleted(ss.Request.Version, ss.Request.Servers, ss.Statuses) {
		return ActionNotifyCompleted, nil
	}

	return ActionNone, nil
}

func NeedUpdate() bool {

}

func NeedReconfigure() bool {

}
