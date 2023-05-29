package dctest

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"log"
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
		keyBytes, err := os.ReadFile(bobPublicKey)

		Expect(err).ShouldNot(HaveOccurred())
		stdout, stderr, err := execAtWithInput(bootServers[0], keyBytes, "etcdpasswd", "cert", "add", user)
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		By("executing command with sudo at boot servers")
		log.Println("bobPrivateKey = ", bobPrivateKey)
		sshKey, err := parsePrivateKey(bobPrivateKey)
		log.Println("sshKey = ", sshKey, " err = ", err)

		Expect(err).ShouldNot(HaveOccurred())
		Eventually(func() error {
			for _, h := range bootServers {
				log.Println("h= ", h, "user = ", user)
				agent, err := sshTo(h, sshKey, user)
				if err != nil {
					log.Println("h= ", h, "  sshTo err = ", err)
					return err
				}
				log.Println("agent = ", agent)
				_, _, err = doExec(agent, nil, "sudo", "ls")
				if err != nil {
					return err
				}
			}
			return nil
		}).Should(Succeed())
	})
}
