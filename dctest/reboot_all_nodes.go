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
		execSafeAt(boot0, "kubectl", "run", "nginx-reboot-test", "--generator=run-pod/v1", "--image=docker.io/nginx:latest")
		execSafeAt(boot0, "kubectl", "run", "debug-reboot-test", "--generator=run-pod/v1", "--image=quay.io/cybozu/ubuntu-debug:18.04", "sleep", "Infinity")
		execSafeAt(boot0, "kubectl", "expose", "pod", "nginx-reboot-test", "--port=80", "--name=nginx-reboot-test")
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
}
