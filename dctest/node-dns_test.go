package dctest

import (
	"encoding/json"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

func testNodeDNS() {
	It("should resolve Service domain names", func() {
		By("getting a Node address")
		pods := &corev1.PodList{}
		stdout := execSafeAt(bootServers[0], "kubectl", "-n", "kube-system", "get", "pods", "-o", "json")
		err := json.Unmarshal(stdout, pods)
		Expect(err).NotTo(HaveOccurred())
		Expect(pods.Items).NotTo(BeEmpty())
		nodeAddr := pods.Items[0].Spec.NodeName

		By("resolving kubernetes.default.svc on " + nodeAddr)
		execSafeAt(bootServers[0], "ckecli", "ssh", "cybozu@"+nodeAddr, "--",
			"curl", "-k", "https://kubernetes.default.svc")
	})

	It("should run kube-proxy on boot servers", func() {
		By("running curl over squid.internet-egress.svc")
		Eventually(func() error {
			_, _, err := execAt(bootServers[0], "env", "https_proxy=http://squid.internet-egress.svc:3128",
				"curl", "-fs", "https://www.cybozu.com/")
			return err
		}).Should(Succeed())
	})
}
