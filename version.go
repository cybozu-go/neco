package neco

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"strings"

	"github.com/cybozu-go/well"
)

var packageName = "neco"

// InstalledNecoVersion get installed neco version
func InstalledNecoVersion(ctx context.Context) (string, error) {
	cmd := well.CommandContext(ctx, "dpkg", "--status", packageName)
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
