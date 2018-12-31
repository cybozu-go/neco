package dctest

import (
	"encoding/json"
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
)

func testSquid() {
	It("should be deployed successfully", func() {
		By("creating k8s resources")
		execSafeAt(boot0, "sed", "-e", "s,%%SQUID_IMAGE%%,$(neco image squid),",
			"/mnt/squid.yml", "|", "kubectl", "create", "-f", "-")

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
				return errors.New("AvailableReplicas is not 2")
			}
			return nil
		}).Should(Succeed())
	})
}
