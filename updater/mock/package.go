package mock

import "context"

// PackageManager is a mock implementation of PackageManager
type PackageManager struct {
	Versions map[string]string
}

// GetVersion returns a version of the name stored in the field
func (m PackageManager) GetVersion(ctx context.Context, name string) (string, error) {
	return m.Versions[name], nil
}
