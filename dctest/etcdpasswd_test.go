package dctest

import (
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// testEtcdpasswd tests etcdpasswd operation
func testEtcdpasswd() {
	It("should be possible to add user", func() {
		By("initialize etcdpasswd")
		user := "bob"

		execRetryAt(bootServers[0], handleNetworkRetry, "etcdpasswd", "set", "start-uid", "2000")
		execRetryAt(bootServers[0], handleNetworkRetry, "etcdpasswd", "set", "start-gid", "2000")
		execRetryAt(bootServers[0], handleNetworkRetry, "etcdpasswd", "set", "default-group", "cybozu")
		execRetryAt(bootServers[0], handleNetworkRetry, "etcdpasswd", "set", "default-groups", "sudo,adm")

		Eventually(func(g Gomega) {
			stdout := execSafeGomegaAt(g, bootServers[0], "etcdpasswd", "user", "list")
			if !strings.Contains(string(stdout), user) {
				execSafeGomegaAt(g, bootServers[0], "etcdpasswd", "user", "add", user)
			}
		}).Should(Succeed())
		execRetryAt(bootServers[0], handleNetworkRetry, "etcdpasswd", "user", "get", user)

		keyBytes, err := os.ReadFile(bobPublicKey)
		Expect(err).ShouldNot(HaveOccurred())

		stdout, stderr, err := execAtWithInput(bootServers[0], keyBytes, "etcdpasswd", "cert", "add", user)
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		By("executing command with sudo at boot servers")
		sshKey, err := parsePrivateKey(bobPrivateKey)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func(g Gomega) {
			for _, h := range bootServers {
				agent, err := sshTo(h, sshKey, user)
				g.Expect(err).ShouldNot(HaveOccurred(), "agent=%v", agent)

				stdout, stderr, err = doExec(agent, nil, "sudo", "ls")
				g.Expect(err).ShouldNot(HaveOccurred(), "agent=%v stdout=%s, stderr=%s", agent, stdout, stderr)
			}
		}).Should(Succeed())
	})
}
