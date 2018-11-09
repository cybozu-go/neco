package storage

import "errors"

var (
	// ErrNotFound is returned when a key is not found in storage.
	ErrNotFound = errors.New("not found")

	// ErrNotStopped is returned when the updater did not stop.
	ErrNotStopped = errors.New("not stopped")

	// ErrNoLeader is returned when the caller lost leadership.
	ErrNoLeader = errors.New("lost leadership")

	// ErrTimedOut is returned when the request is timed out.
	ErrTimedOut = errors.New("timed out")
)
