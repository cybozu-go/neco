package neco

import (
	"os/exec"
)

// GetDebianVersion returns debian package version.
// If "neco" package is not installed, this returns ("", nil).
func GetDebianVersion(pkg string) (string, error) {
	if exec.Command("dpkg", "-s", pkg).Run() != nil {
		return "", nil
	}

	data, err := exec.Command(
		"dpkg-query", "--showformat=${Version}", "-W", pkg).Output()
	if err != nil {
		return "", err
	}

	return string(data), nil
}
