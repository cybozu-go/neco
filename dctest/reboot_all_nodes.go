package dctest

import (
	"encoding/json"
	"fmt"

	"github.com/cybozu-go/sabakan/v2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

// TestRebootAllNodes tests all nodes stop scenario
func TestRebootAllNodes() {
	It("can access a pod from another pod running on different node", func() {
		execSafeAt(boot0, "kubectl", "run", "nginx-reboot-test", "--image=quay.io/cybozu/testhttpd:0", "--generator=run-pod/v1")
		execSafeAt(boot0, "kubectl", "run", "debug-reboot-test", "--generator=run-pod/v1", "--image=quay.io/cybozu/ubuntu-debug:18.04", "sleep", "Infinity")
		execSafeAt(boot0, "kubectl", "expose", "pod", "nginx-reboot-test", "--port=80", "--target-port=8000", "--name=nginx-reboot-test")
		Eventually(func() error {
			_, _, err := execAt(boot0, "kubectl", "exec", "debug-reboot-test", "curl", "http://nginx-reboot-test")
			return err
		}).Should(Succeed())
	})

	It("reboots all nodes", func() {
		stdout, _, err := execAt(boot0, "sabactl", "machines", "get", "--role", "worker")
		Expect(err).ShouldNot(HaveOccurred())
		var machines []sabakan.Machine
		err = json.Unmarshal(stdout, &machines)
		Expect(err).ShouldNot(HaveOccurred())
		for _, m := range machines {
			_, _, err = execAt(boot0, "neco", "ipmipower", "stop", m.Spec.IPv4[0])
			Expect(err).ShouldNot(HaveOccurred())
		}
		for _, m := range machines {
			_, _, err = execAt(boot0, "neco", "ipmipower", "start", m.Spec.IPv4[0])
			Expect(err).ShouldNot(HaveOccurred())
		}
	})

	It("sets all nodes' machine state to healthy", func() {
		Eventually(func() error {
			stdout, _, err := execAt(boot0, "sabactl", "machines", "get", "--role", "worker")
			if err != nil {
				return err
			}

			var machines []sabakan.Machine
			err = json.Unmarshal(stdout, &machines)
			if err != nil {
				return err
			}

			for _, m := range machines {
				stdout := execSafeAt(boot0, "sabactl", "machines", "get-state", m.Spec.Serial)
				state := string(stdout)
				if state != "healthy" {
					return fmt.Errorf("sabakan machine state of %s is not healthy: %s", m.Spec.Serial, state)
				}
			}

			return nil
		}).Should(Succeed())
	})

	It("recovers 5 nodes", func() {
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
			return nil
		}).Should(Succeed())
	})

	It("can access a pod from another pod running on different node, even after rebooting", func() {
		Eventually(func() error {
			_, _, err := execAt(boot0, "kubectl", "exec", "debug-reboot-test", "curl", "http://nginx-reboot-test")
			return err
		}).Should(Succeed())
	})

	It("can pull new image from internet", func() {
		By("deploying a new pod whose image-pull-policy is Always")
		testPod := "test-new-image"
		execSafeAt(boot0, "kubectl", "run", testPod,
			"--image=quay.io/cybozu/testhttpd:0", "--image-pull-policy=Always", "--generator=run-pod/v1")
		By("checking the pod is running")
		Eventually(func() error {
			stdout, _, err := execAt(boot0, "kubectl", "get", "pod", testPod, "-o=json")
			if err != nil {
				return err
			}
			pod := new(corev1.Pod)
			err = json.Unmarshal(stdout, &pod)
			if err != nil {
				return err
			}
			if pod.Status.Phase != "Running" {
				return fmt.Errorf("%s is not Running. current status: %v", testPod, pod.Status)
			}
			return nil
		}).Should(Succeed())

		By("deleting " + testPod)
		execSafeAt(boot0, "kubectl", "delete", "pod", testPod)
	})
}
