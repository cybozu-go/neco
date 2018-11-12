package updater

import (
	"context"
	"reflect"

	"github.com/cybozu-go/neco"
	version "github.com/hashicorp/go-version"
)

type Action int

const (
	ActionNone Action = iota
	ActionNewVersion
	ActionWaitClear
	ActionReconfigure
)

func (s Server) NextAction(ctx context.Context, req *neco.UpdateRequest, rev int64, pkg PackageManager) (Action, error) {
	statuses, err := s.storage.GetStatuses(ctx, rev)
	if err != nil {
		return ActionNone, err
	}
	latest, lrns, err := s.storage.GetInfo(ctx, rev)
	if err != nil {
		return ActionNone, err
	}
	latestVer, err := version.NewVersion(latest)
	if err != nil {
		return ActionNone, err
	}

	current, err := pkg.GetVersion(ctx, neco.NecoPackageName)
	if err != nil {
		return ActionNone, err
	}
	currentVer, err := version.NewVersion(current)
	if err != nil {
		return ActionNone, err
	}

	if req == nil {
		if latestVer.GreaterThan(currentVer) {
			return ActionNewVersion, nil
		}
		return ActionNone, nil
	}

	if req.Stop {
		return ActionWaitClear, nil
	}

	if !reflect.DeepEqual(req.Servers, lrns) {
		return ActionReconfigure, nil
	}

	if req.Version != latest {
		return ActionNewVersion, nil
	}

	if neco.UpdateCompleted(req.Version, req.Servers, statuses) {

	}
}

func NeedUpdate() bool {

}

func NeedReconfigure() bool {

}
