package dctest

import (
	"bytes" // for discovering the cause
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// testEtcdpasswd tests etcdpasswd operation
func testEtcdpasswd() {
	It("should be possible to add user", func() {
		By("initialize etcdpasswd")
		user := "bob"

		// neco #2304 Original code
		//execSafeAt(bootServers[0], "etcdpasswd", "set", "start-uid", "2000")
		//execSafeAt(bootServers[0], "etcdpasswd", "set", "start-gid", "2000")
		//execSafeAt(bootServers[0], "etcdpasswd", "set", "default-group", "cybozu")
		//execSafeAt(bootServers[0], "etcdpasswd", "set", "default-groups", "sudo,adm")
		//execSafeAt(bootServers[0], "etcdpasswd", "user", "add", user)
		//execSafeAt(bootServers[0], "etcdpasswd", "user", "get", user)

		// neco #2304 for discovering the cause
		ret := execSafeAt(bootServers[0], "etcdpasswd", "set", "start-uid", "2000")
		GinkgoWriter.Println("*** etcdpasswd set start-uid,  stdout= ", string(bytes.TrimSpace(ret)))
		ret = execSafeAt(bootServers[0], "etcdpasswd", "set", "start-gid", "2000")
		GinkgoWriter.Println("*** etcdpasswd set start-gid,  stdout= ", string(bytes.TrimSpace(ret)))
		ret = execSafeAt(bootServers[0], "etcdpasswd", "set", "default-group", "cybozu")
		GinkgoWriter.Println("*** etcdpasswd set default-group,  stdout= ", string(bytes.TrimSpace(ret)))
		ret = execSafeAt(bootServers[0], "etcdpasswd", "set", "default-groups", "sudo,adm")
		GinkgoWriter.Println("*** etcdpasswd set default-groups, stdout= ", string(bytes.TrimSpace(ret)))
		ret = execSafeAt(bootServers[0], "etcdpasswd", "user", "add", user)
		GinkgoWriter.Println("*** etcdpasswd user add,  stdout= ", string(bytes.TrimSpace(ret)))
		ret = execSafeAt(bootServers[0], "etcdpasswd", "user", "get", user)
		GinkgoWriter.Println("*** etcdpasswd user get,  stdout= ", string(bytes.TrimSpace(ret)))
		// end

		keyBytes, err := os.ReadFile(bobPublicKey)
		GinkgoWriter.Println("*** keyBytes= ", keyBytes, "err=", err) // neco #2304 for discovering the cause

		Expect(err).ShouldNot(HaveOccurred())
		stdout, stderr, err := execAtWithInput(bootServers[0], keyBytes, "etcdpasswd", "cert", "add", user)
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		By("executing command with sudo at boot servers")
		GinkgoWriter.Println("*** bobPrivateKey= ", bobPrivateKey) // neco #2304 for discovering the cause
		sshKey, err := parsePrivateKey(bobPrivateKey)
		GinkgoWriter.Println("*** sshKey= ", sshKey, " err= ", err) // neco #2304 for discovering the cause

		Expect(err).ShouldNot(HaveOccurred())
		Eventually(func(g Gomega) error {
			for _, h := range bootServers {
				GinkgoWriter.Println("*** h= ", h, "user= ", user) // neco #2304  for discovering the cause
				agent, err := sshTo(h, sshKey, user)
				g.Expect(err).ShouldNot(HaveOccurred(), "agent=%v", agent)
				stdout, stderr, err = doExec(agent, nil, "sudo", "ls")
				g.Expect(err).ShouldNot(HaveOccurred(), "agent=%v stdout=%s, stderr=%s", agent, stdout, stderr)
			}
			return nil
		}).Should(Succeed())
	})
}
