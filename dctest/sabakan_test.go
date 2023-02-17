package dctest

import (
	"github.com/cybozu-go/neco"
	. "github.com/onsi/ginkgo"
	// . "github.com/onsi/gomega"
)

// testSabakan test sabakan
func testSabakan() {
	It("should initialize sabakan", func() {
		By("setting configurations")
		execSafeAt(bootServers[0], "sabactl", "ipam", "set", "-f", neco.SabakanIPAMFile)
		execSafeAt(bootServers[0], "sabactl", "kernel-params", "set", "console=ttyS0")
		execSafeAt(bootServers[0], "sabactl", "machines", "create", "-f", "/mnt/machines.json")
	})
}
