package mock

import "context"

// ReleaseChecker is an implementation of ReleaseChecker
type ReleaseChecker struct {
	Version string
}

// Run runs release cheker server, it block until ctx is canceled
func (c ReleaseChecker) Run(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

// GetLatest returns version specified in the field
func (c ReleaseChecker) GetLatest() string {
	return c.Version
}

// HasUpdate returns always true
func (c ReleaseChecker) HasUpdate() bool {
	return true
}
