package dctest

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// testMachines tests machine control functions.
func testMachines() {
	It("should put BMC/IPMI settings", func() {
		execSafeAt(bootServers[0], "neco", "bmc", "config", "set", "bmc-user", "/mnt/bmc-user.json")
		execSafeAt(bootServers[0], "neco", "bmc", "config", "set", "ipmi-user", "cybozu")
		ipmiUser := execSafeAt(bootServers[0], "neco", "bmc", "config", "get", "ipmi-user")
		Expect(string(ipmiUser)).To(Equal("cybozu\n"))
		execSafeAt(bootServers[0], "neco", "bmc", "config", "set", "ipmi-password", "cybozu")
		ipmiPassword := execSafeAt(bootServers[0], "neco", "bmc", "config", "get", "ipmi-password")
		Expect(string(ipmiPassword)).To(Equal("cybozu\n"))
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
