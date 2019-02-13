package neco

import (
	"context"
	"path/filepath"
	"strings"

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

// RestartService restarts the service simply.
func RestartService(ctx context.Context, name string) error {
	err := well.CommandContext(ctx, "systemctl", "daemon-reload").Run()
	if err != nil {
		return err
	}
	err = well.CommandContext(ctx, "systemctl", "enable", name+".service").Run()
	if err != nil {
		return err
	}
	return well.CommandContext(ctx, "systemctl", "restart", name+".service").Run()
}

// StopService stops the service.
func StopService(ctx context.Context, name string) error {
	return well.CommandContext(ctx, "systemctl", "stop", name+".service").Run()
}

// DisableService disables the service.
func DisableService(ctx context.Context, name string) error {
	return well.CommandContext(ctx, "systemctl", "disable", name+".service").Run()
}

// IsActiveService returns true is the service is active.
func IsActiveService(ctx context.Context, name string) (bool, error) {
	output, err := well.CommandContext(ctx, "systemctl", "is-active", name+".service").Output()
	if err == nil {
		return true, nil
	}
	if strings.TrimSpace(string(output)) == "inactive" {
		return false, nil
	}
	return false, err
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

// StopTimer stops the timer.
func StopTimer(ctx context.Context, name string) error {
	return well.CommandContext(ctx, "systemctl", "stop", name+".timer").Run()
}

// DisableTimer disables the timer.
func DisableTimer(ctx context.Context, name string) error {
	return well.CommandContext(ctx, "systemctl", "disable", name+".timer").Run()
}
