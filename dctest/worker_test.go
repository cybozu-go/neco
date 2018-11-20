package dctest

import (
	"bytes"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func testWorker() {
	var rootToken string
	It("should get root token", func() {
		stdout, _, err := execAt(boot0, "neco", "vault", "show-root-token")
		Expect(err).ShouldNot(HaveOccurred())
		rootToken = string(bytes.TrimSpace(stdout))
		Expect(rootToken).NotTo(BeEmpty())
	})

	It("should success initialize etcdpasswd", func() {
		// wait for vault leader election
		time.Sleep(10 * time.Second)

		execSafeAt(boot0, "neco", "init", "etcdpasswd")

		for _, host := range []string{boot0, boot1, boot2} {
			stdout, stderr, err := execAt(
				host, "sudo", "env", "VAULT_TOKEN="+rootToken, "neco", "init-local", "etcdpasswd")
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

}
