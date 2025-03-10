package dctest

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
)

// testUnbound test installed unbound
func testUnbound() {
	It("should be available", func() {
		By("checking unbound Deployment")
		Eventually(func(g Gomega) {
			deployment := kubectlGetSafe[appsv1.Deployment](g, "--namespace=internet-egress", "deployments/unbound")
			g.Expect(int(deployment.Status.AvailableReplicas)).To(Equal(2), "AvailableReplicas is not 2: %d", int(deployment.Status.AvailableReplicas))
		}).Should(Succeed())

		By("checking PodDisruptionBudget for unbound Deployment")
		pdb := kubectlGetSafe[policyv1.PodDisruptionBudget](Default, "poddisruptionbudgets", "unbound-pdb", "-n", "internet-egress")
		Expect(pdb.Status.CurrentHealthy).Should(Equal(int32(2)))

		checkUnboundExporter("app.kubernetes.io/name=unbound")
	})

	It("should resolve www.cybozu.com", func() {
		By("running a test pod")
		execSafeAt(bootServers[0], "kubectl", "run", "test",
			"--image=$(ckecli images | grep -F ghcr.io/cybozu/unbound:)",
			"--command", "--", "pause")

		By("executing getent hosts www.cybozu.com in test pod")
		Eventually(func() error {
			_, _, err := execAt(bootServers[0], "kubectl", "exec", "test",
				"--", "getent", "hosts", "www.cybozu.com")
			return err
		}).Should(Succeed())

		By("deleting a test pod")
		execSafeAt(bootServers[0], "kubectl", "delete", "pod", "test")
	})
}

func checkUnboundExporter(podLabelSelector string) {
	// should be called in It
	By("checking unbound_exporter")
	Eventually(func(g Gomega) {
		podList := kubectlGetSafe[corev1.PodList](g, "pod", "-n", "internet-egress", "-l", podLabelSelector)
		stdout, stderr, err := execAt(bootServers[0], "curl", "-sSf", "http://"+podList.Items[0].Status.PodIP+":9167/metrics")
		g.Expect(err).NotTo(HaveOccurred(), "stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
		g.Expect(string(stdout)).To(ContainSubstring("unbound_up 1"), "should contain 'unbound_up 1' in the metrics")
		g.Expect(string(stdout)).To(ContainSubstring(`unbound_memory_caches_bytes{cache="message"}`), `should contain 'unbound_memory_caches_bytes{cache="message"}' in the metrics`)
	}).Should(Succeed())
}
