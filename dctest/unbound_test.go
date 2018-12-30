package dctest

import (
	"encoding/json"
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
)

func testUnbound() {
	It("should be deployed as internet-egress/unbound", func() {
		execSafeAt(boot0,
			"sed", "s,%%UNBOUND_IMAGE%%,$(ckecli images | grep quay.io/cybozu/unbound),",
			"/mnt/unbound.yml", "|", "kubectl", "create", "-f", "-")
		Eventually(func() error {
			stdout, _, err := execAt(boot0, "kubectl", "--namespace=internet-egress", "get", "deployments/unbound", "--namespace=kube-system", "-o=json")
			if err != nil {
				return err
			}

			deployment := new(appsv1.Deployment)
			err = json.Unmarshal(stdout, deployment)
			if err != nil {
				return err
			}

			if int(deployment.Status.AvailableReplicas) != 3 {
				return errors.New("AvailableReplicas is not 3")
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
	})
}
