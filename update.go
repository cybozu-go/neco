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
