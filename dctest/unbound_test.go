package dctest

import (
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	policyv1 "k8s.io/api/policy/v1"
)

// testUnbound test installed unbound
func testUnbound() {
	It("should be available", func() {
		By("checking unbound Deployment")
		Eventually(func() error {
			stdout, _, err := execAt(bootServers[0], "kubectl", "--namespace=internet-egress",
				"get", "deployments/unbound", "-o=json")
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
		By("checking PodDisruptionBudget for unbound Deployment")
		pdb := policyv1.PodDisruptionBudget{}
		stdout, stderr, err := execAt(bootServers[0], "kubectl", "get", "poddisruptionbudgets", "unbound-pdb", "-n", "internet-egress", "-o", "json")
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
		err = json.Unmarshal(stdout, &pdb)
		Expect(err).ShouldNot(HaveOccurred(), "data=%s", stdout)
		Expect(pdb.Status.CurrentHealthy).Should(Equal(int32(2)))
	})

	It("should resolve www.cybozu.com", func() {
		By("running a test pod")
		execSafeAt(bootServers[0], "kubectl", "run", "test",
			"--image=$(ckecli images | grep quay.io/cybozu/unbound)",
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
