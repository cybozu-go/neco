package dctest

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

func testNodeDNS() {
	It("should resolve Service domain names", func() {
		By("getting a Node address")
		pods := kubectlGetSafe[corev1.PodList](Default, "-n", "kube-system", "pods")
		Expect(pods.Items).NotTo(BeEmpty())
		nodeAddr := pods.Items[0].Spec.NodeName

		By("resolving kubernetes.default.svc on " + nodeAddr)
		execSafeAt(bootServers[0], "ckecli", "ssh", "cybozu@"+nodeAddr, "--",
			"curl", "-k", "https://kubernetes.default.svc")
	})

	It("should run kube-proxy on boot servers", func() {
		By("running curl over squid.internet-egress.svc")
		Eventually(func(g Gomega) {
			_, _, err := execAt(bootServers[0], "env", "https_proxy=http://squid.internet-egress.svc:3128",
				"curl", "-fs", "https://www.cybozu.com/")
			g.Expect(err).NotTo(HaveOccurred())
		}).Should(Succeed())
	})
}
