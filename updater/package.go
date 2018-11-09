package updater

import (
	"context"
	"os/exec"

	"github.com/cybozu-go/well"
)

// PackageManager is an interface to manage packages
type PackageManager interface {
	GetVersion(ctx context.Context, name string) (string, error)
}

// DebPackageManager is deb package manager
type DebPackageManager struct{}

// GetVersion get a version of the package
func (m DebPackageManager) GetVersion(ctx context.Context, name string) (string, error) {
	if exec.Command("dpkg", "-s", name).Run() != nil {
		return "", nil
	}

	data, err := well.CommandContext(ctx,
		"dpkg-query", "--showformat=${Version}", "-W", name).Output()
	if err != nil {
		return "", err
	}

	return string(data), nil
}
