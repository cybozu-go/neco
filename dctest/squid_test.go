package dctest

import (
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
)

func testSquid() {
	It("should be deployed successfully", func() {
		By("creating k8s resources")
		execSafeAt(boot0, "sed", "-e", "s,%%SQUID_IMAGE%%,$(neco image squid),",
			"-e", "'s,cache_mem .*,cache_mem 200 MB,'",
			"/usr/share/neco/squid.yml", "|", "kubectl", "create", "-f", "-")

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
		By("running nginx pods")
		execSafeAt(boot0, "kubectl", "run", "nginx", "--image=nginx", "--replicas=2")

		Eventually(func() error {
			stdout, _, err := execAt(boot0, "kubectl", "get", "deployments/nginx", "-o=json")
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

		By("removing nginx pods")
		execSafeAt(boot0, "kubectl", "delete", "deployments/nginx")
	})
}
