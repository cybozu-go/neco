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
		// Original code
		//execSafeAt(bootServers[0], "etcdpasswd", "set", "start-uid", "2000")
		//execSafeAt(bootServers[0], "etcdpasswd", "set", "start-gid", "2000")
		//execSafeAt(bootServers[0], "etcdpasswd", "set", "default-group", "cybozu")
		//execSafeAt(bootServers[0], "etcdpasswd", "set", "default-groups", "sudo,adm")
		//execSafeAt(bootServers[0], "etcdpasswd", "user", "add", user)
		//execSafeAt(bootServers[0], "etcdpasswd", "user", "get", user)

		// for discovering the cause
		ret := execSafeAt(bootServers[0], "etcdpasswd", "set", "start-uid", "2000")
		GinkgoWriter.Println("*** etcdpasswd set start-uid = ", string(bytes.TrimSpace(ret)))
		ret = execSafeAt(bootServers[0], "etcdpasswd", "set", "start-gid", "2000")
		GinkgoWriter.Println("*** etcdpasswd set start-gid = ", string(bytes.TrimSpace(ret)))
		ret = execSafeAt(bootServers[0], "etcdpasswd", "set", "default-group", "cybozu")
		GinkgoWriter.Println("*** etcdpasswd set default-group = ", string(bytes.TrimSpace(ret)))
		ret = execSafeAt(bootServers[0], "etcdpasswd", "set", "default-groups", "sudo,adm")
		GinkgoWriter.Println("*** etcdpasswd set default-groups = ", string(bytes.TrimSpace(ret)))
		ret = execSafeAt(bootServers[0], "etcdpasswd", "user", "add", user)
		GinkgoWriter.Println("*** etcdpasswd user add = ", string(bytes.TrimSpace(ret)))
		ret = execSafeAt(bootServers[0], "etcdpasswd", "user", "get", user)
		GinkgoWriter.Println("*** etcdpasswd user get = ", string(bytes.TrimSpace(ret)))

		keyBytes, err := os.ReadFile(bobPublicKey)
		GinkgoWriter.Println("*** keyBytes= ", keyBytes, "err=", err) // for discovering the cause

		Expect(err).ShouldNot(HaveOccurred())
		stdout, stderr, err := execAtWithInput(bootServers[0], keyBytes, "etcdpasswd", "cert", "add", user)
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		By("executing command with sudo at boot servers")
		GinkgoWriter.Println("*** bobPrivateKey= ", bobPrivateKey) // for discovering the cause
		sshKey, err := parsePrivateKey(bobPrivateKey)
		GinkgoWriter.Println("*** sshKey= ", sshKey, " err= ", err) // for discovering the cause

		Expect(err).ShouldNot(HaveOccurred())
		Eventually(func() error {
			for _, h := range bootServers {
				GinkgoWriter.Println("*** h= ", h, "user= ", user) // for discovering the cause
				agent, err := sshTo(h, sshKey, user)
				if err != nil {
					GinkgoWriter.Println("*** h= ", h, "  sshTo err= ", err) // for discovering the cause
					return err
				}
				GinkgoWriter.Println("*** agent= ", agent) // for discovering the cause
				_, _, err = doExec(agent, nil, "sudo", "ls")
				if err != nil {
					GinkgoWriter.Println("*** h= ", h, "  doExec err= ", err) // for discovering the cause
					return err
				}
			}
			return nil
		}).Should(Succeed())
	})
}
