package dctest

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func testCoil() {
	It("should success initialize coil", func() {
		By("coilctl configuration")
		execSafeAt(boot0, "ckecli", "etcd", "user-add", "coil", "/coil/")

		_, stderr, err := execAt(boot0, "ckecli", "etcd", "issue", "coil", "--output", "file")
		Expect(err).ShouldNot(HaveOccurred(), "stderr=%s", stderr)

		_, stderr, err = execAt(boot0, "kubectl", "create", "secret", "generic", "coil-etcd-secrets", "--from-file=etcd-ca.crt", "--from-file=etcd-coil.crt", "--from-file=etcd-coil.key", "-n", "kube-system")
		Expect(err).ShouldNot(HaveOccurred(), "stderr=%s", stderr)

		execSafeAt(boot0, "kubectl", "create", "-f", "/mnt/coil-rbac.yml")
		execSafeAt(boot0, "kubectl", "create", "-f", "/mnt/coil-deploy.yml")
	})
}
