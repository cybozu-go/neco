package dctest

import (
	"fmt"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// testCoilSetup tests Coil setup
func testCoilSetup() {
	It("should be deployed successfully", func() {
		By("waiting for coild DaemonSet and coil-controller Deployment")
		checkCoilNodeDaemonSet()
		checkCoilControllerDeployment()

		Eventually(func(g Gomega) error {
			nl := kubectlGetSafe[corev1.NodeList](g, "nodes")

		OUTER:
			for _, n := range nl.Items {
				for _, cond := range n.Status.Conditions {
					if cond.Type != corev1.NodeReady {
						continue
					}
					g.Expect(cond.Status).To(Equal(corev1.ConditionTrue), "node %s is not ready", n.Name)
					continue OUTER
				}
				return fmt.Errorf("node %s has no readiness status", n.Name)
			}
			return nil
		}).Should(Succeed())

		By("creating IP address pool")
		Eventually(func(g Gomega) {
			data, err := os.ReadFile(addressPoolsFile)
			g.Expect(err).NotTo(HaveOccurred())

			stdout, stderr, err := execAtWithInput(bootServers[0], data, "kubectl", "apply", "-f", "-")
			g.Expect(err).NotTo(HaveOccurred(), "err=%s stdout=%s, stderr=%s", err, stdout, stderr)
		}).Should(Succeed())
	})
}

// testCoil tests Coil
func testCoil() {
	It("should be available", func() {
		By("checking coild DaemonSet and coil-controller Deployment")
		checkCoilNodeDaemonSet()
		checkCoilControllerDeployment()

		// Pod creation is tested somewhere else.
	})
}

func checkCoilNodeDaemonSet() {
	EventuallyWithOffset(1, func(g Gomega) {
		daemonset := kubectlGetSafe[appsv1.DaemonSet](g, "--namespace=kube-system", "daemonsets/coild")
		g.Expect(daemonset.Status.NumberReady).To(Equal(daemonset.Status.DesiredNumberScheduled),
			"NumberReady: %d, DesiredNumberScheduled: %d", daemonset.Status.NumberReady, daemonset.Status.DesiredNumberScheduled)
		g.Expect(int(daemonset.Status.NumberReady)).NotTo(Equal(0), "NumberReady == 0")
	}).Should(Succeed())
}

func checkCoilControllerDeployment() {
	EventuallyWithOffset(1, func(g Gomega) {
		deployment := kubectlGetSafe[appsv1.Deployment](g, "--namespace=kube-system", "deployment/coil-controller")
		g.Expect(int(deployment.Status.ReadyReplicas)).To(Equal(2), "ReadyReplicas should be 2 but %d", deployment.Status.ReadyReplicas)
	}).Should(Succeed())
}
