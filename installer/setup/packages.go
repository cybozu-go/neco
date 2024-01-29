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
		//"fwupd",
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
	err := exec.Command("systemd-detect-virt", "-v", "-q").Run()
	if err != nil {
		args = append(args, "netplan.io")
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

	if err := runCmd("snap", "install", "chromium"); err != nil {
		return fmt.Errorf("failed to install chromium with snap: %w", err)
	}
	return nil
}
