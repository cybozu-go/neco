package dctest

import (
	"github.com/cybozu-go/log"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// TestMachines tests machine control functions.
func TestMachines() {
	It("should register machines to sabakan", func() {
		execSafeAt(boot0, "sabactl", "machines", "create", "-f", "/mnt/machines.json")
	})

	It("should put BMC/IPMI settings", func() {
		execSafeAt(boot0, "neco", "bmc", "config", "set", "bmc-user", "/mnt/bmc-user.json")
		execSafeAt(boot0, "neco", "bmc", "config", "set", "ipmi-user", "cybozu")
		ipmiUser := execSafeAt(boot0, "neco", "bmc", "config", "get", "ipmi-user")
		Expect(string(ipmiUser)).To(Equal("cybozu\n"))
		execSafeAt(boot0, "neco", "bmc", "config", "set", "ipmi-password", "cybozu")
		ipmiPassword := execSafeAt(boot0, "neco", "bmc", "config", "get", "ipmi-password")
		Expect(string(ipmiPassword)).To(Equal("cybozu\n"))
	})

	It("should setup boot server hardware", func() {
		for _, host := range []string{boot0, boot1, boot2} {
			stdout, stderr, err := execAt(host, "sudo", "neco", "bmc", "setup-hw")
			if err != nil {
				log.Error("setup-hw", map[string]interface{}{
					"host":   host,
					"stdout": string(stdout),
					"stderr": string(stderr),
				})
				Expect(err).NotTo(HaveOccurred())
			}
		}
	})
}
