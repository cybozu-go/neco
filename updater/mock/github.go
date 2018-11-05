package mock

import "context"

type GitHub struct {
	Release    string
	PreRelease string
}

func (m GitHub) GetLatestReleaseTag(ctx context.Context) (string, error) {
	return m.Release, nil
}

func (m GitHub) GetLatestPreReleaseTag(ctx context.Context) (string, error) {
	return m.PreRelease, nil
}
