package dctest

import (
	"io/ioutil"

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
		_, _, err = execAtWithInput(boot0, keyBytes, "etcdpasswd", "cert", "add", user)
		Expect(err).ShouldNot(HaveOccurred())

		By("executing command with sudo at boot servers")
		sshKey, err := parsePrivateKey(bobPrivateKey)
		Expect(err).ShouldNot(HaveOccurred())
		Eventually(func() error {
			for _, h := range []string{boot0, boot1, boot2} {
				agent, err := sshTo(h, sshKey, user)
				if err != nil {
					return err
				}
				_, _, err = doExec(agent, nil, "sudo", "ls")
				if err != nil {
					return err
				}
			}
			return nil
		}).Should(Succeed())
	})
}
