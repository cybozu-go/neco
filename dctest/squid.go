package dctest

import (
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
)

// TestSquid test installed squid
func TestSquid() {
	It("should be available", func() {
		By("checking squid Deployment")
		Eventually(func() error {
			stdout, _, err := execAt(boot0, "kubectl", "--namespace=internet-egress",
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
	})

	It("should serve for docker daemon", func() {
		By("running testhttpd pods")
		execSafeAt(boot0, "kubectl", "run", "testhttpd", "--image=quay.io/cybozu/testhttpd:0", "--replicas=2")

		Eventually(func() error {
			stdout, _, err := execAt(boot0, "kubectl", "get", "deployments/testhttpd", "-o=json")
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

		By("removing testhttpd deployments")
		execSafeAt(boot0, "kubectl", "delete", "deployments/testhttpd")
	})

	It("reverts proxy config", func() {
		execSafeAt(boot0, "neco", "config", "set", "proxy", proxy)
	})
}
