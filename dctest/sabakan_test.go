package dctest

import (
	"github.com/cybozu-go/neco"
	. "github.com/onsi/ginkgo/v2"
	// . "github.com/onsi/gomega"
)

// testSabakan test sabakan
func testSabakan() {
	It("should initialize sabakan", func() {
		By("setting configurations")
		execSafeAt(bootServers[0], "sabactl", "ipam", "set", "-f", neco.SabakanIPAMFile)
		execRetryAt(bootServers[0], handleNetworkRetry, "sabactl", "kernel-params", "set", "console=ttyS0")
		execSafeAt(bootServers[0], "sabactl", "machines", "create", "-f", "/mnt/machines.json")
	})
}
