package neco

import (
	"context"
	"path/filepath"

	"github.com/cybozu-go/well"
)

// ServiceFile returns the filesystem path of a systemd service.
func ServiceFile(name string) string {
	return filepath.Join(systemdDir, name+".service")
}

// TimerFile returns the filesystem path of a systemd timer.
func TimerFile(name string) string {
	return filepath.Join(systemdDir, name+".timer")
}

// StartService does following:
// 1. systemctl daemon-reload
// 2. systemctl enable NAME.service
// 3. systemctl start NAME.service
func StartService(ctx context.Context, name string) error {
	return startUnit(ctx, name, "service")
}

// StartTimer does following:
// 1. systemctl daemon-reload
// 2. systemctl enable NAME.timer
// 3. systemctl start NAME.timer
func StartTimer(ctx context.Context, name string) error {
	return startUnit(ctx, name, "timer")
}

func startUnit(ctx context.Context, name, unit string) error {
	err := well.CommandContext(ctx, "systemctl", "daemon-reload").Run()
	if err != nil {
		return err
	}
	err = well.CommandContext(ctx, "systemctl", "enable", name+"."+unit).Run()
	if err != nil {
		return err
	}
	return well.CommandContext(ctx, "systemctl", "start", name+"."+unit).Run()
}
