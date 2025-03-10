package main

import (
	"fmt"
	"os"
	"os/exec"
)

var (
	purgeList = []string{
		"apport",
		"apport-symptoms",
		"fwupd",
		"nano",
		//"netplan.io",
		//"popularity-contest",
		"unattended-upgrades",
		"update-manager-core",
		"ubuntu-release-upgrader-core",
	}

	installList = []string{
		"jq",
		"ipvsadm",
	}
)

func purgePackages() error {
	fmt.Fprintln(os.Stderr, "Purging packages...")
	args := append([]string{"purge", "-y", "--autoremove"}, purgeList...)

	// cloud-init depends on netplan.io, so it can be purged
	// only from non-virtual machines.
	// The popularity-contest package is not included in the cloud image, but is included in the iso image.
	// Therefore, it is necessary to purge them in a non-vm environment.
	err := exec.Command("systemd-detect-virt", "-v", "-q").Run()
	if err != nil {
		args = append(args, "netplan.io", "popularity-contest")
	}

	return runCmd("apt-get", args...)
}

func installPackages(pkgs ...string) error {
	fmt.Fprintf(os.Stderr, "Installing %v...\n", pkgs)
	args := append([]string{
		"install", "-y", "--no-install-recommends",
	}, pkgs...)
	return runCmd("apt-get", args...)
}

func installChromium() error {
	err := exec.Command("systemd-detect-virt", "-v", "-q").Run()
	if err == nil {
		return nil
	}

	fmt.Fprintln(os.Stderr, "Installing chromium...")
	if err := runCmd("snap", "set", "system", "proxy.http="+config.proxy); err != nil {
		return fmt.Errorf("failed to set proxy for snap: %w", err)
	}
	if err := runCmd("snap", "set", "system", "proxy.https="+config.proxy); err != nil {
		return fmt.Errorf("failed to set proxy for snap: %w", err)
	}

	if err := runCmd("snap", "refresh", "snapd"); err != nil {
		return fmt.Errorf("failed to update snapd with snap: %w", err)
	}
	retryCount := 0
	for {
		err := runCmd("snap", "install", "chromium")
		retryCount++
		if err != nil {
			if retryCount > 3 {
				return fmt.Errorf("failed to install chromium with snap: %w", err)
			}
		} else {
			break
		}
	}
	return nil
}
