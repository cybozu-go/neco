package worker

import (
	"github.com/cybozu-go/neco"
)

// UpdateAborted returns true if the current update process was aborted.
//
// The condition is checked as follows:
// 1. If the status is not for the current version, it is ignored.
// 2. If the status has Cond==CondComplete, it is ignored.
// 3. If the status has Cond==CondAbort, this returns true.
// 4. If the status has Step==1 for other workers, it is ignored.
// 5. Otherwise, the process was aborted.
func UpdateAborted(version string, mylrn int, statuses map[int]*neco.UpdateStatus) bool {
	for lrn, u := range statuses {
		if u.Version != version {
			continue
		}
		switch u.Cond {
		case neco.CondComplete:
			continue
		case neco.CondAbort:
			return true
		}
		if u.Step == 1 && lrn != mylrn {
			continue
		}
		return true
	}

	return false
}
