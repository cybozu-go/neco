// +build linux

package serf

import (
	"bufio"
	"errors"
	"io/ioutil"
	"os"
	"strings"
)

var serialPath = "/sys/class/dmi/id/product_serial"
var osReleasePath = "/etc/os-release"

// GetOSVersionID returns value of VERSION_ID in /etc/os-release
func GetOSVersionID() (string, error) {
	f, err := os.Open(osReleasePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	for s := bufio.NewScanner(f); s.Scan(); {
		kv := strings.Split(s.Text(), "=")
		if len(kv) != 2 || strings.TrimSpace(kv[0]) != "VERSION_ID" {
			continue
		}
		return strings.Trim(kv[1], "\""), nil
	}
	return "", errors.New("failed to get os version")
}

// GetSerial returns serial number
func GetSerial() (string, error) {
	serial, err := ioutil.ReadFile(serialPath)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(serial)), nil
}
