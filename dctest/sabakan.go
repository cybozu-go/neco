package dctest

import (
	. "github.com/onsi/ginkgo"
	// . "github.com/onsi/gomega"
)

// TestSabakan test sabakan
func TestSabakan() {
	It("should initialize sabakan", func() {
		By("setting configurations")
		execSafeAt(boot0, "sabactl", "ipam", "set", "-f", "/etc/neco/sabakan_ipam.json")
		execSafeAt(boot0, "sabactl", "kernel-params", "set", "console=ttyS0")
		execSafeAt(boot0, "sabactl", "machines", "create", "-f", "/mnt/machines.json")
	})
}
