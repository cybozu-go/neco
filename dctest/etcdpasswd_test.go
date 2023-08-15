package dctest

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
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

		keyBytes, err := os.ReadFile(bobPublicKey)

		Expect(err).ShouldNot(HaveOccurred())
		stdout, stderr, err := execAtWithInput(bootServers[0], keyBytes, "etcdpasswd", "cert", "add", user)
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		By("executing command with sudo at boot servers")
		sshKey, err := parsePrivateKey(bobPrivateKey)

		Expect(err).ShouldNot(HaveOccurred())
		Eventually(func(g Gomega) error {
			for _, h := range bootServers {
				agent, err := sshTo(h, sshKey, user)
				g.Expect(err).ShouldNot(HaveOccurred(), "agent=%v", agent)
				stdout, stderr, err = doExec(agent, nil, "sudo", "ls")
				g.Expect(err).ShouldNot(HaveOccurred(), "agent=%v stdout=%s, stderr=%s", agent, stdout, stderr)
			}
			return nil
		}).Should(Succeed())
	})
}
