package neco

import (
	"context"
	"path/filepath"

	"github.com/cybozu-go/well"
)

// ServiceFile returns the filesystem path of a service.
func ServiceFile(name string) string {
	return filepath.Join(systemdDir, name+".service")
}

// StartService does following:
// 1. systemctl daemon-reload
// 2. systemctl enable NAME.service
// 3. systemctl start NAME.service
func StartService(ctx context.Context, name string) error {
	err := well.CommandContext(ctx, "systemctl", "daemon-reload").Run()
	if err != nil {
		return err
	}
	err = well.CommandContext(ctx, "systemctl", "enable", name+".service").Run()
	if err != nil {
		return err
	}
	return well.CommandContext(ctx, "systemctl", "start", name+".service").Run()
}
