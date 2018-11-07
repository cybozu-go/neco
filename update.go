package neco

import "time"

// UpdateRequest represents request from neco-updater
type UpdateRequest struct {
	Version   string    `json:"version"`
	Servers   []int     `json:"servers"`
	Stop      bool      `json:"stop"`
	StartedAt time.Time `json:"started_at"`
}

// IsMember returns true if a boot server is the member of this update request.
func (r UpdateRequest) IsMember(lrn int) bool {
	for _, n := range r.Servers {
		if n == lrn {
			return true
		}
	}

	return false
}

// UpdateCondition is the condition of the update process.
type UpdateCondition int

// Possible update conditions.
const (
	CondRunning = iota
	CondAbort
	CondComplete
)

// UpdateStatus represents status report from neco-worker
type UpdateStatus struct {
	Version string          `json:"version"`
	Step    int             `json:"step"`
	Cond    UpdateCondition `json:"cond"`
	Message string          `json:"message"`
}

// UpdateAborted returns true if the current update process was aborted.
//
// The condition is checked as follows:
// 1. If the status is not for the current version, it is ignored.
// 2. If the status has Cond==CondComplete, it is ignored.
// 3. If the status has Cond==CondAbort, this returns true.
// 4. If the status has Step==1 for other workers, it is ignored.
// 5. Otherwise, the process was aborted.
func UpdateAborted(version string, mylrn int, statuses map[int]*UpdateStatus) bool {
	for lrn, u := range statuses {
		if u.Version != version {
			continue
		}
		switch u.Cond {
		case CondComplete:
			continue
		case CondAbort:
			return true
		}
		if u.Step == 1 && lrn != mylrn {
			continue
		}
		return true
	}

	return false
}

// UpdateCompleted returns true if the current update process has
// completed successfully.
func UpdateCompleted(version string, lrns []int, statuses map[int]*UpdateStatus) bool {
	for _, lrn := range lrns {
		st, ok := statuses[lrn]
		if !ok {
			return false
		}

		if st.Version != version {
			return false
		}

		if st.Cond != CondComplete {
			return false
		}
	}

	return true
}
