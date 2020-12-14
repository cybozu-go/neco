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
		"nano",
		"netplan.io",
		"popularity-contest",
		"unattended-upgrades",
		"update-manager-core",
	}

	installList = []string{
		"freeipmi-tools",
		"jq",
	}
)

func purgePackages() error {
	fmt.Fprintln(os.Stderr, "Purging packages...")
	args := append([]string{"purge", "-y", "--autoremove"}, purgeList...)
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

	if err := runCmd("snap", "install", "chromium"); err != nil {
		return fmt.Errorf("failed to install chromium with snap: %w", err)
	}
	return nil
}