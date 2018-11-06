package neco

import "time"

// UpdateRequest represents request from neco-updater
type UpdateRequest struct {
	Version   string    `json:"version"`
	Servers   []int     `json:"servers"`
	Stop      bool      `json:"stop"`
	StartedAt time.Time `json:"started_at"`
}

// UpdateStatus represents status report from neco-worker
type UpdateStatus struct {
	Version  string `json:"version"`
	Step     int    `json:"step"`
	Finished bool   `json:"finished"`
	Error    bool   `json:"error"`
	Message  string `json:"message"`
}

// UpdateAborted returns true if the current update process was aborted.
//
// The condition is checked as follows:
// 1. If the status is not for the current version, it is ignored.
// 2. If the status has Finished==true, it is ignored.
// 3. If the status has Step > 1, the update process was aborted at that Step.
// 4. If the status has Error==true, the update process was aborted at step 1.
func UpdateAborted(version string, statuses map[int]*UpdateStatus) bool {
	for _, u := range statuses {
		if u.Version != version {
			continue
		}
		if u.Finished {
			continue
		}
		if u.Step > 1 || u.Error {
			return true
		}
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

		if !st.Finished {
			return false
		}
	}

	return true
}
