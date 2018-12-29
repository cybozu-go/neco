package dctest

import (
	"encoding/json"
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func testCoil() {
	It("should success initialize coil", func() {
		By("coilctl configuration")
		execSafeAt(boot0, "ckecli", "etcd", "user-add", "coil", "/coil/")

		_, stderr, err := execAt(boot0, "ckecli", "etcd", "issue", "coil", "--output", "file")
		Expect(err).ShouldNot(HaveOccurred(), "stderr=%s", stderr)

		_, stderr, err = execAt(boot0, "kubectl", "create", "secret", "generic", "coil-etcd-secrets", "--from-file=etcd-ca.crt", "--from-file=etcd-coil.crt", "--from-file=etcd-coil.key", "-n", "kube-system")
		Expect(err).ShouldNot(HaveOccurred(), "stderr=%s", stderr)

		execSafeAt(boot0, "kubectl", "create", "-f", "/mnt/coil-rbac.yml")
		execSafeAt(boot0, "kubectl", "create", "-f", "/mnt/coil-deploy.yml")

		Eventually(func() error {
			stdout, _, err := execAt(boot0, "kubectl", "get", "daemonsets/coil-node", "--namespace=kube-system", "-o=json")
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

		By("create IP address pool")
		stdout, stderr, err := execAt(boot0, "kubectl", "--namespace=kube-system", "get", "pods", "--selector=k8s-app=coil-controllers", "-o=json")
		Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		podList := new(corev1.PodList)
		err = json.Unmarshal(stdout, podList)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(podList.Items)).To(Equal(1))
		podName := podList.Items[0].Name

		_, _, err = execAt(boot0, "test", "-f", "/tmp/coil-pool-create-done")
		if err != nil {
			_, stderr, err = execAt(boot0, "kubectl", "--namespace=kube-system", "exec", podName, "/coilctl", "pool", "create", "default", "10.64.0.0/14", "4")
			Expect(err).NotTo(HaveOccurred(), "stderr=%s", stderr)
			_, stderr, err = execAt(boot0, "kubectl", "create", "namespace", "dmz")
			Expect(err).NotTo(HaveOccurred(), "stderr=%s", stderr)
			_, stderr, err = execAt(boot0, "kubectl", "--namespace=kube-system", "exec", podName, "/coilctl", "pool", "create", "dmz", "172.17.0.0/26", "0")
			Expect(err).NotTo(HaveOccurred(), "stderr=%s", stderr)
		}
	})
}
