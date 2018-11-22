package neco

import "os/exec"

// HardwareType represents
type HardwareType int

// known hardware types
const (
	HWTypeNil HardwareType = iota
	HWTypePlacematVM
	HWTypeDell
)

// DetectHardware detects hardware type.
func DetectHardware() (HardwareType, error) {
	vm, err := IsVM()
	if err != nil {
		return HWTypeNil, err
	}

	if vm {
		return HWTypePlacematVM, nil
	}
	return HWTypeDell, nil
}

// IsVM returns true if executed on VM.
func IsVM() (bool, error) {
	err := exec.Command("systemd-detect-virt", "-v", "-q").Run()
	if err == nil {
		return true, nil
	}
	if _, ok := err.(*exec.ExitError); ok {
		// command started successfully, and returned non-zero to say "this is not VM"
		return false, nil
	}
	// command invocation failed
	return false, err
}
