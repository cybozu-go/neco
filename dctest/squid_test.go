package dctest

import (
	"encoding/json"
	"fmt"

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
		Eventually(func() error {
			stdout, _, err := execAt(bootServers[0], "kubectl", "--namespace=internet-egress",
				"get", "deployments/squid", "-o=json")
			if err != nil {
				return err
			}

			deployment := new(appsv1.Deployment)
			err = json.Unmarshal(stdout, deployment)
			if err != nil {
				return err
			}

			if int(deployment.Status.AvailableReplicas) != 2 {
				return fmt.Errorf("AvailableReplicas is not 2: %d", int(deployment.Status.AvailableReplicas))
			}
			return nil
		}).Should(Succeed())
		By("checking PodDisruptionBudget for squid Deployment")
		pdb := policyv1.PodDisruptionBudget{}
		stdout, stderr, err := execAt(bootServers[0], "kubectl", "get", "poddisruptionbudgets", "squid-pdb", "-n", "internet-egress", "-o", "json")
		if err != nil {
			Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
		}
		err = json.Unmarshal(stdout, &pdb)
		Expect(err).ShouldNot(HaveOccurred(), "data=%s", stdout)
		Expect(pdb.Status.CurrentHealthy).Should(Equal(int32(2)))

		checkUnboundExporter("app.kubernetes.io/name=squid")
	})

	It("should serve for docker daemon", func() {
		By("running a testhttpd pod")
		execSafeAt(bootServers[0], "kubectl", "run", "testhttpd", "--image=quay.io/cybozu/testhttpd:0")

		Eventually(func() error {
			stdout, _, err := execAt(bootServers[0], "kubectl", "get", "pod/testhttpd", "-o=json")
			if err != nil {
				return err
			}

			pod := new(corev1.Pod)
			err = json.Unmarshal(stdout, pod)
			if err != nil {
				return err
			}

			if pod.Status.Phase != corev1.PodRunning {
				return fmt.Errorf("Pod is not running: %s", pod.Status.Phase)
			}
			return nil
		}).Should(Succeed())

		By("removing the testhttpd pod")
		execSafeAt(bootServers[0], "kubectl", "delete", "pod/testhttpd")
	})
}
