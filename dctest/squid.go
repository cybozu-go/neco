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
		By("running busybox pods")
		execSafeAt(boot0, "kubectl", "run", "busybox", "--image=docker.io/busybox:latest", "--replicas=2",
			`--overrides='{"spec":{"template":{"spec":{"securityContext":{"runAsUser":10000}}}}}'`,
			"--", "httpd", "-f", "-p", "8000", "-h", "/etc")

		Eventually(func() error {
			stdout, _, err := execAt(boot0, "kubectl", "get", "deployments/busybox", "-o=json")
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

		By("removing busybox pods")
		execSafeAt(boot0, "kubectl", "delete", "deployments/busybox")
	})
}
