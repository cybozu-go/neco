package dctest

import (
	"bytes"
	"context"
	"fmt"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/hashicorp/vault/api"
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
		cfg := api.DefaultConfig()
		cfg.Address = fmt.Sprintf("https://%s:8200", neco.BootNode0IP(0).String())
		vc, err := api.NewClient(cfg)
		Expect(err).ShouldNot(HaveOccurred())

		err = neco.WaitVaultLeader(context.Background(), vc)
		Expect(err).ShouldNot(HaveOccurred())

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
