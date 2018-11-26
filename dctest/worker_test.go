package dctest

import (
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func testWorker() {
	It("should success initialize etcdpasswd", func() {
		// wait for vault leader election
		time.Sleep(10 * time.Second)

		token := getVaultToken()

		execSafeAt(boot0, "neco", "init", "etcdpasswd")

		for _, host := range []string{boot0, boot1, boot2} {
			stdout, stderr, err := execAt(
				host, "sudo", "env", "VAULT_TOKEN="+token, "neco", "init-local", "etcdpasswd")
			if err != nil {
				log.Error("neco init-local etcdpasswd", map[string]interface{}{
					"host":   host,
					"stdout": string(stdout),
					"stderr": string(stderr),
				})
				Expect(err).ShouldNot(HaveOccurred())
			}
			execSafeAt(host, "test", "-f", neco.EtcdpasswdConfFile)
			execSafeAt(host, "test", "-f", neco.EtcdpasswdKeyFile)
			execSafeAt(host, "test", "-f", neco.EtcdpasswdCertFile)

			execSafeAt(host, "systemctl", "-q", "is-active", "ep-agent.service")
		}
	})

	It("should success initialize Serf", func() {
		for _, host := range []string{boot0, boot1, boot2} {
			execSafeAt(host, "test", "-f", neco.SerfConfFile)
			execSafeAt(host, "test", "-x", neco.SerfHandler)
			execSafeAt(host, "systemctl", "-q", "is-active", "serf.service")
		}
	})

	It("should success initialize sabakan", func() {
		token := getVaultToken()

		execSafeAt(boot0, "neco", "init", "sabakan")

		for _, host := range []string{boot0, boot1, boot2} {
			stdout, stderr, err := execAt(
				host, "sudo", "env", "VAULT_TOKEN="+token, "neco", "init-local", "sabakan")
			if err != nil {
				log.Error("neco init-local sabakan", map[string]interface{}{
					"host":   host,
					"stdout": string(stdout),
					"stderr": string(stderr),
				})
				Expect(err).ShouldNot(HaveOccurred())
			}
			execSafeAt(host, "test", "-d", neco.SabakanDataDir)
			execSafeAt(host, "test", "-f", neco.SabakanConfFile)
			execSafeAt(host, "test", "-f", neco.SabakanKeyFile)
			execSafeAt(host, "test", "-f", neco.SabakanCertFile)

			execSafeAt(host, "systemctl", "-q", "is-active", "sabakan.service")
		}
	})

	It("should success initialize sabakan data", func() {
		execSafeAt(boot0, "sabactl", "ipam", "set", "-f", "/mnt/ipam.json")
		execSafeAt(boot0, "sabactl", "dhcp", "set", "-f", "/mnt/dhcp.json")
		execSafeAt(boot0, "sabactl", "machines", "create", "-f", "/mnt/machines.json")
	})

	It("should update machine state in sabakan", func() {
		// Restart serf after machine registered to update state in sabakan
		for _, host := range []string{boot0, boot1, boot2} {
			execSafeAt(host, "sudo", "systemctl", "restart", "serf.service")
		}

		for _, ip := range []string{"10.69.0.3", "10.69.0.195", "10.69.1.131"} {
			Eventually(func() string {
				return execSafeAt(boot0, "sabactl", "machines", "get", "-ipv4", ip)
			}).Should(ContainSubstring(`"state": "healthy"`))
		}
	})

	It("should success initialize cke", func() {
		token := getVaultToken()

		execSafeAt(boot0, "neco", "init", "cke")

		for _, host := range []string{boot0, boot1, boot2} {
			stdout, stderr, err := execAt(
				host, "sudo", "env", "VAULT_TOKEN="+token, "neco", "init-local", "cke")
			if err != nil {
				log.Error("neco init-local cke", map[string]interface{}{
					"host":   host,
					"stdout": string(stdout),
					"stderr": string(stderr),
				})
				Expect(err).ShouldNot(HaveOccurred())
			}
			execSafeAt(host, "test", "-f", neco.CKEConfFile)
			execSafeAt(host, "test", "-f", neco.CKEKeyFile)
			execSafeAt(host, "test", "-f", neco.CKECertFile)

			execSafeAt(host, "systemctl", "-q", "is-active", "cke.service")
		}
	})

	It("should success retrieve cke leader", func() {
		stdout := execSafeAt(boot0, "ckecli", "leader")
		Expect(stdout).To(ContainSubstring("boot-"))
	})

	It("should setup hardware", func() {
		for _, host := range []string{boot0, boot1, boot2} {
			stdout, stderr, err := execAt(host, "sudo", "setup-hw")
			if err != nil {
				log.Error("setup-hw", map[string]interface{}{
					"host":   host,
					"stdout": string(stdout),
					"stderr": string(stderr),
				})
				Expect(err).ShouldNot(HaveOccurred())
			}
		}
	})
}
