package dctest

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
)

// testSquid test installed squid
func testSquid() {
	It("should be available", func() {
		By("checking squid Deployment")
		Eventually(func(g Gomega) {
			deployment := kubectlGetSafe[appsv1.Deployment](g, "--namespace=internet-egress", "deployments/squid")
			g.Expect(int(deployment.Status.AvailableReplicas)).To(Equal(3), "AvailableReplicas is not 3: %d", int(deployment.Status.AvailableReplicas))
		}).Should(Succeed())

		By("checking PodDisruptionBudget for squid Deployment")
		pdb := kubectlGetSafe[policyv1.PodDisruptionBudget](Default, "poddisruptionbudgets", "squid-pdb", "-n", "internet-egress")
		Expect(int(pdb.Status.CurrentHealthy)).Should(Equal(3))

		checkUnboundExporter("app.kubernetes.io/name=squid")
		checkSquidExporter("app.kubernetes.io/name=squid")
	})

	It("should serve for docker daemon", func() {
		By("running a testhttpd pod")
		execSafeAt(bootServers[0], "kubectl", "run", "testhttpd", "--image=ghcr.io/cybozu/testhttpd:0")

		Eventually(func(g Gomega) {
			pod := kubectlGetSafe[corev1.Pod](g, "pod/testhttpd")
			g.Expect(pod.Status.Phase).To(Equal(corev1.PodRunning), "Pod is not running: %s", pod.Status.Phase)
		}).Should(Succeed())

		By("removing the testhttpd pod")
		execSafeAt(bootServers[0], "kubectl", "delete", "pod/testhttpd")
	})
}

func checkSquidExporter(podLabelSelector string) {
	By("checking squid_exporter")
	Eventually(func(g Gomega) {
		podList := kubectlGetSafe[corev1.PodList](g, "pod", "-n", "internet-egress", "-l", podLabelSelector)
		stdout, stderr, err := execAt(bootServers[0], "curl", "-sSf", "http://"+podList.Items[0].Status.PodIP+":9100/metrics")
		g.Expect(err).NotTo(HaveOccurred(), "stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
		g.Expect(string(stdout)).To(ContainSubstring("cpu_time_total"), "should contain 'cpu_time_total' in the metrics")
		g.Expect(string(stdout)).To(ContainSubstring(`squid_service_times_http_requests_all{percentile="5", duration_minutes="5"}`),
			`should contain 'squid_service_times_http_requests_all{percentile="5", duration_minutes="5"}' in the metrics`)
	}).Should(Succeed())
}
