package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
)

func logerror(err error, msg string) {
	fmt.Fprintf(os.Stderr, "error: %s: %v\n", msg, err)
}

func runCmd(cmd string, args ...string) error {
	c := exec.Command(cmd, args...)
	c.Env = []string{
		"http_proxy=" + config.proxy,
		"https_proxy=" + config.proxy,
		"DEBIAN_FRONTEND=noninteractive",
	}
	out, err := c.CombinedOutput()
	if err != nil {
		logerror(err, string(out))
	}
	return err
}

func systemctl(args ...string) error {
	return runCmd("systemctl", args...)
}

func enableService(name string) error {
	if err := systemctl("daemon-reload"); err != nil {
		return err
	}
	if err := systemctl("enable", name+".service"); err != nil {
		return err
	}
	return systemctl("restart", name+".service")
}

func installService(name string, unit string) error {
	err := ioutil.WriteFile(filepath.Join("/etc/systemd/system", name+".service"), []byte(unit), 0644)
	if err != nil {
		return fmt.Errorf("failed to create service file for %s: %w", name, err)
	}
	return enableService(name)
}

func installOverrideConf(name string, data string) error {
	err := os.Mkdir(filepath.Join("/etc/systemd/system", name+".service.d"), 0755)
	if err != nil {
		return fmt.Errorf("failed to mkdir /etc/systemd/system/%s.service.d: %w", name, err)
	}
	err = ioutil.WriteFile(filepath.Join("/etc/systemd/system", name+".service.d", "override.conf"), []byte(data), 0644)
	if err != nil {
		return fmt.Errorf("failed to create override.conf for %s: %w", name, err)
	}
	return nil
}
