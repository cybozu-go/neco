package dctest

import (
	"github.com/cybozu-go/log"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestMachines() {
	It("should register machines to sabakan", func() {
		execSafeAt(boot0, "sabactl", "machines", "create", "-f", "/mnt/machines.json")
	})

	It("should put BMC/IPMI settings", func() {
		// neco bmc config set bmc-user bmc-user.json
		// neco bmc config set ipam-user cybozu
		// neco bmc config set ipam-password cybozu
	})

	It("should setup boot server hardware", func() {
		for _, host := range []string{boot0, boot1, boot2} {
			stdout, stderr, err := execAt(host, "echo", "sudo", "neco", "bmc", "setup-hw")
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
