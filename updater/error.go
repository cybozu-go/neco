package updater

import "errors"

// Retriable errors in neco-updator
var (
	ErrNoReleases   = errors.New("no neco packages are released")
	ErrNoMembers    = errors.New("no boot servers are joined")
	ErrUpdateFailed = errors.New("update failed on worker(s)")
)
