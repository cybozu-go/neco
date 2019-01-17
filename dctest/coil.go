package dctest

import (
	"encoding/json"
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// TestCoil tests Coil
func TestCoil() {
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

		By("creating k8s resources")
		execSafeAt(boot0, "kubectl", "create", "-f", "/usr/share/neco/coil-rbac.yml")
		execSafeAt(boot0, "sed", "s,%%COIL_IMAGE%%,$(neco image coil),",
			"/usr/share/neco/coil-deploy.yml", "|", "kubectl", "create", "-f", "-")

		Eventually(func() error {
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

			if int(daemonset.Status.NumberReady) != 5 {
				return errors.New("NumberReady is not 5")
			}
			return nil
		}).Should(Succeed())

		By("creating IP address pool")
		stdout, stderr, err := execAt(boot0, "kubectl", "--namespace=kube-system", "get", "pods", "--selector=k8s-app=coil-controllers", "-o=json")
		Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		podList := new(corev1.PodList)
		err = json.Unmarshal(stdout, podList)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(podList.Items)).To(Equal(1))
		podName := podList.Items[0].Name

		_, stderr, err = execAt(boot0, "kubectl", "--namespace=kube-system", "exec", podName, "/coilctl", "pool", "create", "default", "10.64.0.0/14", "5")
		Expect(err).NotTo(HaveOccurred(), "stderr=%s", stderr)
		_, stderr, err = execAt(boot0, "kubectl", "create", "namespace", "internet-egress")
		Expect(err).NotTo(HaveOccurred(), "stderr=%s", stderr)
		_, stderr, err = execAt(boot0, "kubectl", "--namespace=kube-system", "exec", podName, "/coilctl", "pool", "create", "internet-egress", "172.17.0.0/28", "0")
		Expect(err).NotTo(HaveOccurred(), "stderr=%s", stderr)
	})
}
