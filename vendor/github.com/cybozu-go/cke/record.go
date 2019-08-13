package cke

import (
	"time"
)

// RecordStatus is status of an operation
type RecordStatus string

// Record statuses
const (
	StatusNew       = RecordStatus("new")
	StatusRunning   = RecordStatus("running")
	StatusCancelled = RecordStatus("cancelled")
	StatusCompleted = RecordStatus("completed")
)

// Record represents a record of an operation
type Record struct {
	ID        int64        `json:"id,string"`
	Status    RecordStatus `json:"status"`
	Operation string       `json:"operation"`
	Command   Command      `json:"command"`
	Targets   []string     `json:"targets"`
	Error     string       `json:"error"`
	StartAt   time.Time    `json:"start-at"`
	EndAt     time.Time    `json:"end-at"`
}

// NewRecord creates new `Record`
func NewRecord(id int64, op string, targets []string) *Record {
	return &Record{
		ID:        id,
		Status:    StatusNew,
		Operation: op,
		Targets:   targets,
		StartAt:   time.Now().UTC(),
	}
}

// Cancel cancels the operation
func (r *Record) Cancel() {
	r.Status = StatusCancelled
	r.EndAt = time.Now().UTC()
}

// Complete completes the operation
func (r *Record) Complete() {
	r.Status = StatusCompleted
	r.EndAt = time.Now().UTC()
}

// SetCommand updates the record for the new command
func (r *Record) SetCommand(c Command) {
	r.Status = StatusRunning
	r.Command = c
}

// SetError cancels the operation with error information
func (r *Record) SetError(e error) {
	r.Status = StatusCancelled
	r.Error = e.Error()
	r.EndAt = time.Now().UTC()
}
