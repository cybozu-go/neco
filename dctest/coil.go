package dctest

import (
	"encoding/json"
	"errors"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// TestCoilSetup tests Coil setup
func TestCoilSetup() {
	It("should be deployed successfully", func() {
		By("preparing etcd user and certificates")
		execSafeAt(boot0, "ckecli", "etcd", "user-add", "coil", "/coil/")

		_, stderr, err := execAt(boot0, "ckecli", "etcd", "issue", "coil", "--output", "file")
		Expect(err).ShouldNot(HaveOccurred(), "stderr=%s", stderr)

		_, stderr, err = execAt(boot0, "kubectl", "--namespace=kube-system", "create", "secret",
			"generic", "coil-etcd-secrets",
			"--from-file=etcd-ca.crt",
			"--from-file=etcd-coil.crt",
			"--from-file=etcd-coil.key")
		Expect(err).ShouldNot(HaveOccurred(), "stderr=%s", stderr)

		By("waiting for coil-node DaemonSet and coil-controllers Deployment")
		checkCoilNodeDaemonSet()
		checkCoilControllersDeployment()

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

		By("waiting for kube-system/cke-etcd getting created")
		Eventually(func() error {
			_, _, err := execAt(boot0, "kubectl", "--namespace=kube-system", "get", "endpoints/cke-etcd")
			return err
		}).Should(Succeed())

		By("creating IP address pool")
		stdout, stderr, err := execAt(boot0, "kubectl", "--namespace=kube-system", "get", "pods", "--selector=app.kubernetes.io/name=coil-controllers", "-o=json")
		Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		podList := new(corev1.PodList)
		err = json.Unmarshal(stdout, podList)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(podList.Items)).To(Equal(1))
		podName := podList.Items[0].Name

		_, stderr, err = execAt(boot0, "kubectl", "--namespace=kube-system", "exec", podName, "/coilctl", "pool", "create", "default", "10.64.0.0/14", "5")
		Expect(err).NotTo(HaveOccurred(), "stderr=%s", stderr)
		_, stderr, err = execAt(boot0, "kubectl", "--namespace=kube-system", "exec", podName, "/coilctl", "pool", "create", "internet-egress", "172.17.0.0/28", "0")
		Expect(err).NotTo(HaveOccurred(), "stderr=%s", stderr)
	})
}

// TestCoil tests Coil
func TestCoil() {
	It("should be available", func() {
		By("checking coil-node DaemonSet and coil-controllers Deployment")
		checkCoilNodeDaemonSet()
		checkCoilControllersDeployment()

		By("listing pools")
		stdout, stderr, err := execAt(boot0, "kubectl", "--namespace=kube-system", "get", "pods", "--selector=app.kubernetes.io/name=coil-controllers", "-o=json")
		Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		podList := new(corev1.PodList)
		err = json.Unmarshal(stdout, podList)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(podList.Items)).To(Equal(1))
		podName := podList.Items[0].Name

		stdout, stderr, err = execAt(boot0, "kubectl", "--namespace=kube-system", "exec", podName, "/coilctl", "pool", "list")
		Expect(err).NotTo(HaveOccurred(), "stderr=%s", stderr)
		Expect(stdout).To(ContainSubstring("default:"))
		Expect(stdout).To(ContainSubstring("internet-egress:"))
	})
}

func checkCoilNodeDaemonSet() {
	EventuallyWithOffset(1, func() error {
		stdout, _, err := execAt(boot0, "kubectl", "--namespace=kube-system",
			"get", "daemonsets/coil-node", "-o=json")
		if err != nil {
			return err
		}

		daemonset := new(appsv1.DaemonSet)
		err = json.Unmarshal(stdout, daemonset)
		if err != nil {
			return err
		}

		if int(daemonset.Status.NumberReady) != 6 {
			return errors.New("NumberReady is not 6")
		}
		return nil
	}).Should(Succeed())
}

func checkCoilControllersDeployment() {
	EventuallyWithOffset(1, func() error {
		stdout, _, err := execAt(boot0, "kubectl", "--namespace=kube-system",
			"get", "deployment/coil-controllers", "-o=json")
		if err != nil {
			return err
		}

		deployment := new(appsv1.Deployment)
		err = json.Unmarshal(stdout, deployment)
		if err != nil {
			return err
		}

		if int(deployment.Status.AvailableReplicas) != 1 {
			return errors.New("AvailableReplicas is not 1")
		}
		return nil
	}).Should(Succeed())
}
