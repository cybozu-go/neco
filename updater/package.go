package updater

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"strings"

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
	cmd := well.CommandContext(ctx, "dpkg", "--status", name)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	s := bufio.NewScanner(bytes.NewReader(output))
	for s.Scan() {
		line := s.Text()
		if strings.HasPrefix(line, "Version:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Version:")), nil

		}
	}
	return "", errors.New("No 'Version:' field by dpkg --status")
}
