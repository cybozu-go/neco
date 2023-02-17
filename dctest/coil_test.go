package dctest

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	. "github.com/onsi/ginkgo"
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

		Eventually(func() error {
			stdout, _, err := execAt(bootServers[0], "kubectl", "get", "nodes", "-o", "json")
			if err != nil {
				return err
			}

			var nl corev1.NodeList
			err = json.Unmarshal(stdout, &nl)
			if err != nil {
				return err
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

		By("creating IP address pool")
		data, err := os.ReadFile(addressPoolsFile)
		Expect(err).NotTo(HaveOccurred())
		stdout, stderr, err := execAtWithInput(bootServers[0], data, "kubectl", "apply", "-f", "-")
		Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
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
	EventuallyWithOffset(1, func() error {
		stdout, _, err := execAt(bootServers[0], "kubectl", "--namespace=kube-system",
			"get", "daemonsets/coild", "-o=json")
		if err != nil {
			return err
		}

		daemonset := new(appsv1.DaemonSet)
		err = json.Unmarshal(stdout, daemonset)
		if err != nil {
			return err
		}

		if daemonset.Status.NumberReady != daemonset.Status.DesiredNumberScheduled {
			return fmt.Errorf("NumberReady: %d, DesiredNumberScheduled: %d", daemonset.Status.NumberReady, daemonset.Status.DesiredNumberScheduled)
		}
		if daemonset.Status.NumberReady == 0 {
			return errors.New("NumberReady == 0")
		}
		return nil
	}).Should(Succeed())
}

func checkCoilControllerDeployment() {
	EventuallyWithOffset(1, func() error {
		stdout, _, err := execAt(bootServers[0], "kubectl", "--namespace=kube-system",
			"get", "deployment/coil-controller", "-o=json")
		if err != nil {
			return err
		}

		deployment := new(appsv1.Deployment)
		err = json.Unmarshal(stdout, deployment)
		if err != nil {
			return err
		}

		if deployment.Status.ReadyReplicas != 2 {
			return fmt.Errorf("ReadyReplicas is not 2 but %d", deployment.Status.ReadyReplicas)
		}
		return nil
	}).Should(Succeed())
}
