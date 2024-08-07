package dctest

import (
	"bytes"
	"fmt"
	"time"

	"github.com/cybozu-go/neco"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// testInit test initialization steps
func testInit() {
	It("should reset failed status of systemd-networkd-wait-online.service", func() {
		// systemd-networkd-wait-online.service may fail due to timeout on dctest env
		// Frankly speaking, it is not a problem because the test program can already connect to boot servers.
		for _, host := range bootServers {
			execSafeAt(host, "sudo", "systemctl", "reset-failed", "systemd-networkd-wait-online.service")
		}
	})

	It("should create a Vault admin user", func() {
		// wait for vault leader election
		time.Sleep(10 * time.Second)

		stdout, stderr, err := execAt(bootServers[0], "neco", "vault", "show-root-token")
		Expect(err).ShouldNot(HaveOccurred(), "stderr=%s", stderr)
		token := string(bytes.TrimSpace(stdout))

		execSafeAt(bootServers[0], "env", "VAULT_TOKEN="+token, "vault", "auth", "enable",
			"-default-lease-ttl=2h", "-max-lease-ttl=24h", "userpass")
		execSafeAt(bootServers[0], "env", "VAULT_TOKEN="+token, "vault", "write",
			"auth/userpass/users/admin", "policies=admin,ca-admin", "password=cybozu")
		execSafeAt(bootServers[0], "env", "VAULT_TOKEN="+token, "vault", "token", "revoke", "-self")
	})

	It("should success initialize etcdpasswd", func() {
		token := getVaultToken()

		execSafeAt(bootServers[0], "neco", "init", "etcdpasswd")

		for _, host := range bootServers {
			stdout, stderr, err := execAt(
				host, "sudo", "env", "VAULT_TOKEN="+token, "neco", "init-local", "etcdpasswd")
			Expect(err).ShouldNot(HaveOccurred(), "host=%s, stdout=%s, stderr=%s", host, stdout, stderr)

			execSafeAt(host, "test", "-f", neco.EtcdpasswdConfFile)
			execSafeAt(host, "test", "-f", neco.EtcdpasswdKeyFile)
			execSafeAt(host, "test", "-f", neco.EtcdpasswdCertFile)

			execSafeAt(host, "systemctl", "-q", "is-active", "ep-agent.service")
		}
	})

	It("should initialize teleport", func() {
		execSafeAt(bootServers[0], "neco", "init", "teleport")
	})

	It("should success initialize Serf", func() {
		for _, host := range bootServers {
			execSafeAt(host, "test", "-f", neco.SerfConfFile)
			execSafeAt(host, "systemctl", "-q", "is-active", "serf.service")
		}
	})

	It("should success initialize setup-serf-tags", func() {
		for _, host := range bootServers {
			execSafeAt(host, "test", "-f", "/usr/local/bin/setup-serf-tags")
			execSafeAt(host, "systemctl", "-q", "is-active", "setup-serf-tags.timer")
		}
		By("getting systemd unit statuses by serf members")
		Eventually(func() error {
			m, err := getSerfBootMembers()
			if err != nil {
				return err
			}
			// Number of boot servers is 3
			if len(m.Members) != 3 {
				return fmt.Errorf("too few boot servers: %d", len(m.Members))
			}
			for _, member := range m.Members {
				tag, ok := member.Tags["systemd-units-failed"]
				if !ok {
					return fmt.Errorf("member %s does not define tag systemd-units-failed", member.Name)
				}
				if tag != "" {
					return fmt.Errorf("member %s fails systemd units: %s", member.Name, tag)
				}
			}
			return nil
		}).Should(Succeed())
	})

	It("should success initialize sabakan", func() {
		token := getVaultToken()

		execSafeAt(bootServers[0], "neco", "init", "sabakan")

		for _, host := range bootServers {
			stdout, stderr, err := execAt(
				host, "sudo", "env", "VAULT_TOKEN="+token, "neco", "init-local", "sabakan")
			Expect(err).ShouldNot(HaveOccurred(), "host=%s, stdout=%s, stderr=%s", host, stdout, stderr)

			execSafeAt(host, "test", "-d", neco.SabakanDataDir)
			execSafeAt(host, "test", "-f", neco.SabakanConfFile)
			execSafeAt(host, "test", "-f", neco.SabakanEtcdKeyFile)
			execSafeAt(host, "test", "-f", neco.SabakanEtcdCertFile)
			execSafeAt(host, "test", "-f", neco.SabactlBashCompletionFile)

			Eventually(func(g Gomega) {
				stdout, stderr, err := execAt(host, "systemctl", "-q", "is-active", "sabakan.service")
				g.Expect(err).NotTo(HaveOccurred(), "host=%s, stdout=%s, stderr=%s", host, stdout, stderr)

				stdout, stderr, err = execAt(host, "systemctl", "-q", "is-active", "sabakan-state-setter.service")
				g.Expect(err).NotTo(HaveOccurred(), "host=%s, stdout=%s, stderr=%s", host, stdout, stderr)
			}).Should(Succeed())
		}
	})

	It("should success initialize cke", func() {
		token := getVaultToken()

		By("initializing etcd for CKE")
		execSafeAt(bootServers[0], "neco", "init", "cke")

		for _, host := range bootServers {
			stdout, stderr, err := execAt(
				host, "sudo", "env", "VAULT_TOKEN="+token, "neco", "init-local", "cke")
			Expect(err).ShouldNot(HaveOccurred(), "host=%s, stdout=%s, stderr=%s", host, stdout, stderr)

			execSafeAt(host, "test", "-f", neco.CKEConfFile)
			execSafeAt(host, "test", "-f", neco.CKEKeyFile)
			execSafeAt(host, "test", "-f", neco.CKECertFile)

			execSafeAt(host, "systemctl", "-q", "is-active", "cke.service")
		}

		By("initializing Vault for CKE")
		execSafeAt(bootServers[0], "env", "VAULT_TOKEN="+token, "ckecli", "vault", "init")
	})

	It("should success retrieve cke leader", func() {
		stdout := execSafeAt(bootServers[0], "ckecli", "leader")
		Expect(stdout).To(ContainSubstring("boot-"))
	})

	It("should success initialize neco-rebooter", func() {
		for _, host := range bootServers {
			Eventually(func(g Gomega) {
				stdout, stderr, err := execAt(host, "test", "-f", neco.NecoRebooterConfFile)
				g.Expect(err).NotTo(HaveOccurred(), "host=%s, stdout=%s, stderr=%s", host, stdout, stderr)
				stdout, stderr, err = execAt(host, "systemctl", "-q", "is-active", "neco-rebooter.service")
				g.Expect(err).NotTo(HaveOccurred(), "host=%s, stdout=%s, stderr=%s", host, stdout, stderr)
			}).Should(Succeed())
		}
	})

	It("should generate SSH key for worker nodes", func() {
		execSafeAt(bootServers[0], "neco", "ssh", "generate")
	})
}
