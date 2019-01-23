package dctest

import (
	"io/ioutil"
	"path/filepath"

	"github.com/cybozu-go/log"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// TestEtcdpasswd tests etcdpasswd operation
func TestEtcdpasswd() {
	It("should be possible to add user", func() {
		By("initialize etcdpasswd")
		user := "bob"
		execSafeAt(boot0, "etcdpasswd", "set", "start-uid", "2000")
		execSafeAt(boot0, "etcdpasswd", "set", "start-gid", "2000")
		execSafeAt(boot0, "etcdpasswd", "set", "default-group", "cybozu")
		execSafeAt(boot0, "etcdpasswd", "set", "default-groups", "sudo,adm")
		execSafeAt(boot0, "etcdpasswd", "user", "add", user)
		execSafeAt(boot0, "etcdpasswd", "user", "get", user)
		keyBytes, err := ioutil.ReadFile(bobPublicKey)
		Expect(err).ShouldNot(HaveOccurred())
		publicKeyPath := filepath.Join("/tmp", filepath.Base(bobPublicKey))
		_, _, err = execAtWithInput(boot0, keyBytes, "sudo", "tee", publicKeyPath)
		Expect(err).ShouldNot(HaveOccurred())
		execSafeAt(boot0, "etcdpasswd", "cert", "add", user, publicKeyPath)
		execSafeAt(boot0, "rm", publicKeyPath)

		By("executing command with sudo at boot servers")
		sshKey, err := parsePrivateKey(bobPrivateKey)
		Expect(err).ShouldNot(HaveOccurred())
		hosts := []string{boot0, boot1, boot2}
		Eventually(func() error {
			for _, h := range hosts {
				agent, err := sshTo(h, sshKey, "bob")
				if err != nil {
					return err
				}
				_, stderr, err := doExec(agent, nil, "sudo", "ls")
				if err != nil {
					log.Error("failed to exec sudo ls", map[string]interface{}{
						log.FnError: stderr,
					})
					return err
				}
			}
			return nil
		}).Should(Succeed())
	})
}
