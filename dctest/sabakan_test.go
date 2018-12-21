package dctest

import (
	. "github.com/onsi/ginkgo"
	// . "github.com/onsi/gomega"
)

func testSabakan() {
	It("should initialize sabakan", func() {
		By("setting configrations")
		execSafeAt(boot0, "sabactl", "ipam", "set", "-f", "/mnt/ipam.json")
		execSafeAt(boot0, "sabactl", "dhcp", "set", "-f", "/mnt/dhcp.json")
		execSafeAt(boot0, "sabactl", "machines", "create", "-f", "/mnt/machines.json")
		execSafeAt(boot0, "sabactl", "kernel-params", "set", "console=ttyS0")

		// test sabakan's behavior
	})
}
