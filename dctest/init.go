package dctest

import (
	"bytes"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// TestInit test initialization steps
func TestInit() {
	It("should create a Vault admin user", func() {
		// wait for vault leader election
		time.Sleep(10 * time.Second)

		stdout, stderr, err := execAt(boot0, "neco", "vault", "show-root-token")
		Expect(err).ShouldNot(HaveOccurred(), "stderr=%s", stderr)
		token := string(bytes.TrimSpace(stdout))

		execSafeAt(boot0, "env", "VAULT_TOKEN="+token, "vault", "auth", "enable",
			"-default-lease-ttl=2h", "-max-lease-ttl=24h", "userpass")
		execSafeAt(boot0, "env", "VAULT_TOKEN="+token, "vault", "write",
			"auth/userpass/users/admin", "policies=admin,ca-admin", "password=cybozu")
		execSafeAt(boot0, "env", "VAULT_TOKEN="+token, "vault", "token", "revoke", "-self")
	})

	It("should success initialize etcdpasswd", func() {
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
				Expect(err).NotTo(HaveOccurred())
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
				Expect(err).NotTo(HaveOccurred())
			}
			execSafeAt(host, "test", "-d", neco.SabakanDataDir)
			execSafeAt(host, "test", "-f", neco.SabakanConfFile)
			execSafeAt(host, "test", "-f", neco.SabakanKeyFile)
			execSafeAt(host, "test", "-f", neco.SabakanCertFile)

			execSafeAt(host, "systemctl", "-q", "is-active", "sabakan.service")
		}
	})

	It("should success initialize cke", func() {
		token := getVaultToken()

		By("initializing etcd for CKE")
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
				Expect(err).NotTo(HaveOccurred())
			}
			execSafeAt(host, "test", "-f", neco.CKEConfFile)
			execSafeAt(host, "test", "-f", neco.CKEKeyFile)
			execSafeAt(host, "test", "-f", neco.CKECertFile)

			execSafeAt(host, "systemctl", "-q", "is-active", "cke.service")
		}

		By("initializing Vault for CKE")
		execSafeAt(boot0, "env", "VAULT_TOKEN="+token, "ckecli", "vault", "init")
	})

	It("should success retrieve cke leader", func() {
		stdout := execSafeAt(boot0, "ckecli", "leader")
		Expect(stdout).To(ContainSubstring("boot-"))
	})

	It("should generate SSH key for worker nodes", func() {
		execSafeAt(boot0, "neco", "ssh", "generate")
	})
}
