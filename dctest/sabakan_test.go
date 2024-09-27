package dctest

import (
	"strings"

	"github.com/cybozu-go/neco"
	. "github.com/onsi/ginkgo/v2"
)

func handleSabakanInternalServerError(stdout, stderr string, err error) bool {
	return strings.Contains(stderr, "Internal Server Error")
}

// testSabakan test sabakan
func testSabakan() {
	It("should initialize sabakan", func() {
		By("setting configurations")
		execSafeAt(bootServers[0], "sabactl", "ipam", "set", "-f", neco.SabakanIPAMFile)
		execRetryAt(bootServers[0], handleSabakanInternalServerError, "sabactl", "kernel-params", "set", "console=ttyS0")
		execRetryAt(bootServers[0], handleSabakanInternalServerError, "sabactl", "machines", "create", "-f", "/mnt/machines.json")
	})
}
