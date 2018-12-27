package dctest

import (
	"encoding/json"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

func testCKE() {
	It("should generates cluster.yml automatically", func() {
		vaultToken := getVaultToken()

		By("initialize Vault for CKE")
		execSafeAt(boot0, "env", "VAULT_TOKEN="+vaultToken, "ckecli", "vault", "init")

		By("generating SSH key for worker nodes")
		execSafeAt(boot0, "neco", "ssh", "generate")

		By("regenerating ignitions and registering machines")
		execSafeAt(boot0, "sabactl", "ignitions", "delete", "worker", "0.0.2")
		execSafeAt(boot0, "neco", "sabakan-upload", "--ignitions-only")
		execSafeAt(boot0, "sabactl", "machines", "create", "-f", "/mnt/machines.json")

		By("setting configurations")
		execSafeAt(boot0, "ckecli", "constraints", "set", "control-plane-count", "3")
		execSafeAt(boot0, "ckecli", "constraints", "set", "minimum-workers", "2")
		execSafeAt(boot0, "ckecli", "sabakan", "set-template", "/mnt/cke-template.yml")
		execSafeAt(boot0, "ckecli", "sabakan", "set-url", "http://localhost:10080")

		By("waiting for cluster.yml generation")
		Eventually(func() error {
			_, _, err := execAt(boot0, "ckecli", "cluster", "get")
			return err
		}, 20*time.Minute).Should(Succeed())
	})

	It("wait for Kubernetes cluster to become ready", func() {
		By("generating kubeconfig for cluster admin")
		execSafeAt(boot0, "mkdir", ".kube")
		execSafeAt(boot0, "ckecli", "kubernetes", "issue", ">", ".kube/config")

		By("waiting nodes")
		Eventually(func() error {
			stdout, _, err := execAt(boot0, "kubectl", "get", "nodes", "-o", "json")
			if err != nil {
				return err
			}

			var nl corev1.NodeList
			err = json.Unmarshal(stdout, &nl)
			if err != nil {
				return err
			}

			if len(nl.Items) != 5 {
				return fmt.Errorf("too few nodes: %d", len(nl.Items))
			}

		OUTER:
			for _, n := range nl.Items {
				for _, cond := range n.Status.Conditions {
					if cond.Type != corev1.NodeReady {
						continue
					}
					if cond.Status != corev1.ConditionTrue {
						return fmt.Errorf("node %s is not ready", n.Name)
					}
					continue OUTER
				}

				return fmt.Errorf("node %s has no readiness status", n.Name)
			}

			return nil
		}).Should(Succeed())
	})
}
