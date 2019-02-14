package neco

import (
	"os/exec"
)

// GetDebianVersion returns debian package version.
func GetDebianVersion(pkg string) (string, error) {
	err := exec.Command("dpkg", "-s", pkg).Run()
	if err != nil {
		return "", err
	}

	data, err := exec.Command(
		"dpkg-query", "--showformat=${Version}", "-W", pkg).Output()
	if err != nil {
		return "", err
	}

	return string(data), nil
}
