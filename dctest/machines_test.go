package dctest

import (
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func handleNetworkRetry(stdout, stderr string, err error) bool {
	return strings.Contains(stderr, "missing HTTP content-type") ||
		strings.Contains(stderr, "etcdserver: request timed out")
}

// testMachines tests machine control functions.
func testMachines() {
	It("should put BMC/IPMI settings", func() {
		// test set/get functions
		execRetryAt(bootServers[0], handleNetworkRetry, "neco", "bmc", "config", "set", "bmc-user", "/mnt/bmc-user.json")

		execRetryAt(bootServers[0], handleNetworkRetry, "neco", "bmc", "config", "set", "ipmi-user", "cybozu1")
		ipmiUser := execSafeAt(bootServers[0], "neco", "bmc", "config", "get", "ipmi-user")
		Expect(string(ipmiUser)).To(Equal("cybozu1\n"))
		execRetryAt(bootServers[0], handleNetworkRetry, "neco", "bmc", "config", "set", "ipmi-password", "cybozu2")
		ipmiPassword := execSafeAt(bootServers[0], "neco", "bmc", "config", "get", "ipmi-password")
		Expect(string(ipmiPassword)).To(Equal("cybozu2\n"))

		execRetryAt(bootServers[0], handleNetworkRetry, "neco", "bmc", "config", "set", "repair-user", "cybozu3")
		repairUser := execSafeAt(bootServers[0], "neco", "bmc", "config", "get", "repair-user")
		Expect(string(repairUser)).To(Equal("cybozu3\n"))
		execRetryAt(bootServers[0], handleNetworkRetry, "neco", "bmc", "config", "set", "repair-password", "cybozu4")
		repairPassword := execSafeAt(bootServers[0], "neco", "bmc", "config", "get", "repair-password")
		Expect(string(repairPassword)).To(Equal("cybozu4\n"))

		// finally set user/password hard-coded in underlying virtual BMCs
		execRetryAt(bootServers[0], handleNetworkRetry, "neco", "bmc", "config", "set", "ipmi-user", "cybozu")
		execRetryAt(bootServers[0], handleNetworkRetry, "neco", "bmc", "config", "set", "ipmi-password", "cybozu")
	})

	It("should setup boot server hardware", func() {
		for _, host := range bootServers {
			// "neco bmc setup-hw" occasionally fails due to unstable startup of sabakan, so use Eventually()
			Eventually(func() error {
				stdout, stderr, err := execAt(host, "sudo", "neco", "bmc", "setup-hw")
				if err != nil {
					return fmt.Errorf("neco bmc setup-hw failed; host: %s, err: %s, stdout: %s, stderr: %s", host, err, stdout, stderr)
				}
				return nil
			}).Should(Succeed())
		}
	})
}
