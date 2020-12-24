package neco

import (
	"bytes"
	"errors"
	"io/ioutil"
	"strconv"
	"strings"
)

// MyLRN returns logical rack number of own node
func MyLRN() (int, error) {
	data, err := ioutil.ReadFile(RackFile)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(string(bytes.TrimSpace(data)))
}

// MyCluster returns cluster name of own node
func MyCluster() (string, error) {
	data, err := ioutil.ReadFile(ClusterFile)
	if err != nil {
		return "", err
	}
	return string(bytes.TrimSpace(data)), nil
}

// OSCodename returns the OS release codename of the host.
// See man os-release
// e.g. bionic, focal
func OSCodename() (string, error) {
	data, err := ioutil.ReadFile("/etc/os-release")
	if err != nil {
		return "", err
	}
	for _, l := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(l, "VERSION_CODENAME=") {
			return l[len("VERSION_CODENAME="):], nil
		}
	}

	return "", errors.New("no VERSION_CODENAME in /etc/os-release")
}
