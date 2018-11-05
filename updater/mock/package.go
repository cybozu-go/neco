package mock

import "context"

type PackageManager struct {
	Versions map[string]string
}

func (m PackageManager) GetVersion(ctx context.Context, name string) (string, error) {
	return m.Versions[name], nil
}
