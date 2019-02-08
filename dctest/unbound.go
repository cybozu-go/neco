package dctest

import (
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
)

// TestUnboundSetup test unbound setup
func TestUnboundSetup() {
	It("should be deployed as internet-egress/unbound", func() {
		execSafeAt(boot0,
			"sed", "s,%%UNBOUND_IMAGE%%,$(ckecli images | grep quay.io/cybozu/unbound),",
			"/usr/share/neco/unbound.yml", "|", "kubectl", "create", "-f", "-")
	})
}

// TestUnbound test installed unbound
func TestUnbound() {
	It("should be available", func() {
		By("checking unbound Deployment")
		Eventually(func() error {
			stdout, _, err := execAt(boot0, "kubectl", "--namespace=internet-egress",
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
	})

	It("should resolve www.cybozu.com", func() {
		By("running a test pod")
		execSafeAt(boot0, "kubectl", "run", "test",
			"--image=$(ckecli images | grep quay.io/cybozu/cke-tools)",
			"--generator=run-pod/v1", "--", "/bin/sleep", "infinity")

		By("executing getent hosts www.cybozu.com in test pod")
		Eventually(func() error {
			_, _, err := execAt(boot0, "kubectl", "exec", "test",
				"getent", "hosts", "www.cybozu.com")
			return err
		}).Should(Succeed())

		By("deleting a test pod")
		execSafeAt(boot0, "kubectl", "delete", "pod", "test")
	})
}
