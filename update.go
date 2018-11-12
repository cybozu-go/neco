package neco

import "time"

// Environments to use release or pre-release neco
const (
	StagingEnv = "staging"
	ProdEnv    = "prod"
)

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
	CondNotRunning = iota
	CondRunning
	CondAbort
	CondComplete
)

// String implements io.Stringer
func (c UpdateCondition) String() string {
	switch c {
	case CondNotRunning:
		return "not running"
	case CondRunning:
		return "running"
	case CondAbort:
		return "aborted"
	case CondComplete:
		return "completed"
	default:
		return "unknown"
	}
}

// UpdateStatus represents status report from neco-worker
type UpdateStatus struct {
	Version string          `json:"version"`
	Step    int             `json:"step"`
	Cond    UpdateCondition `json:"cond"`
	Message string          `json:"message"`
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
