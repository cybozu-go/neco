package neco

import (
	"errors"
	"io/ioutil"
	"os/exec"
	"strings"
)

// HardwareType represents
type HardwareType int

// hardware type
const (
	HWTypeNil HardwareType = iota
	HWTypeVM
	HWTypeContainer
	HWTypeDell
)

// DetectHardware detects hardware type.
func DetectHardware() (HardwareType, error) {
	t, err := systemdDetectVirt()
	if err != nil {
		return HWTypeNil, err
	}
	switch t {
	case "qemu", "kvm", "zvm", "vmware", "microsoft", "oracle", "xen", "bochs", "uml", "parallels", "bhyve":
		// VM
		return HWTypeVM, nil
	case "openvz", "lxc", "lxc-libvirt", "systemd-nspawn", "docker", "rkt":
		// Container
		return HWTypeContainer, nil
	case "none":
	default:
		return HWTypeNil, errors.New("unsupported hardware type: " + t)
	}

	vendorBytes, err := ioutil.ReadFile("/sys/class/dmi/id/chassis_vendor")
	if err != nil {
		return HWTypeNil, err
	}
	vendor := strings.TrimSpace(string(vendorBytes))

	switch vendor {
	case "Dell Inc.":
		return HWTypeDell, nil
	}
	return HWTypeNil, errors.New("unsupported hardware vendor: " + vendor)
}

func systemdDetectVirt() (string, error) {
	out, err := exec.Command("systemd-detect-virt").Output()
	t := strings.TrimSpace(string(out))
	if err != nil && t != "none" {
		return "", err
	}
	return t, nil
}
