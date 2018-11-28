package sabakan

import "regexp"

var (
	reValidKernelParams = regexp.MustCompile(`^[0-9a-zA-Z.,-_= ]*$`)
)

// IsValidKernelParams returns true if s is valid as an kernel params
func IsValidKernelParams(s string) bool {
	return reValidKernelParams.MatchString(s)
}

// KernelParams is a kernel parameters.
type KernelParams string
