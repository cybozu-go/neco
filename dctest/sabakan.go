package dctest

import (
	. "github.com/onsi/ginkgo"
	// . "github.com/onsi/gomega"
)

// TestSabakan test sabakan
func TestSabakan() {
	It("should initialize sabakan", func() {
		By("setting configrations")
		execSafeAt(boot0, "sabactl", "ipam", "set", "-f", "/mnt/ipam.json")
		execSafeAt(boot0, "sabactl", "dhcp", "set", "-f", "/mnt/dhcp.json")
		execSafeAt(boot0, "sabactl", "kernel-params", "set", "console=ttyS0")

		// machines will be registered later in cke_test.go.
	})
}
