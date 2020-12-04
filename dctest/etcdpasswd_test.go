package dctest

import (
	"io/ioutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// testEtcdpasswd tests etcdpasswd operation
func testEtcdpasswd() {
	It("should be possible to add user", func() {
		By("initialize etcdpasswd")
		user := "bob"
		execSafeAt(bootServers[0], "etcdpasswd", "set", "start-uid", "2000")
		execSafeAt(bootServers[0], "etcdpasswd", "set", "start-gid", "2000")
		execSafeAt(bootServers[0], "etcdpasswd", "set", "default-group", "cybozu")
		execSafeAt(bootServers[0], "etcdpasswd", "set", "default-groups", "sudo,adm")
		execSafeAt(bootServers[0], "etcdpasswd", "user", "add", user)
		execSafeAt(bootServers[0], "etcdpasswd", "user", "get", user)
		keyBytes, err := ioutil.ReadFile(bobPublicKey)
		Expect(err).ShouldNot(HaveOccurred())
		_, _, err = execAtWithInput(bootServers[0], keyBytes, "etcdpasswd", "cert", "add", user)
		Expect(err).ShouldNot(HaveOccurred())

		By("executing command with sudo at boot servers")
		sshKey, err := parsePrivateKey(bobPrivateKey)
		Expect(err).ShouldNot(HaveOccurred())
		Eventually(func() error {
			for _, h := range bootServers {
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
