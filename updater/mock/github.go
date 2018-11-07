package mock

import "context"

// GitHub is a mock implementation
type GitHub struct {
	Release    string
	PreRelease string
}

// GetLatestReleaseTag returns a release version in the field
func (m GitHub) GetLatestReleaseTag(ctx context.Context) (string, error) {
	return m.Release, nil
}

// GetLatestPreReleaseTag returns a pre-release version in the field
func (m GitHub) GetLatestPreReleaseTag(ctx context.Context) (string, error) {
	return m.PreRelease, nil
}
