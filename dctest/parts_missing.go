package dctest

import (
	"encoding/json"
	"errors"

	"github.com/cybozu-go/sabakan/v2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

// TestPartsMissing test parts missing scenario
func TestPartsMissing() {
	var targetIP string

	It("transitition machine state to unhealthy", func() {
		stdout, _, err := execAt(boot0, "kubectl", "get", "nodes", "-o", "json")
		Expect(err).ShouldNot(HaveOccurred())

		var nl corev1.NodeList
		err = json.Unmarshal(stdout, &nl)
		Expect(err).ShouldNot(HaveOccurred())
		targetIP = nl.Items[0].Name

		By("copying dummy redfish data to " + targetIP)
		Eventually(func() error {
			return copyDummyMissingRedfishDataToWorker(targetIP)
		}).Should(Succeed())

		By("checking machine state")
		Eventually(func() error {
			stdout, _, err := execAt(boot0, "sabactl", "machines", "get", "--ipv4", targetIP)
			if err != nil {
				return err
			}
			var machines []sabakan.Machine
			err = json.Unmarshal(stdout, &machines)
			if err != nil {
				return err
			}
			for _, m := range machines {
				if m.Status.State.String() != "unhealthy" {
					return errors.New(m.Spec.Serial + " is not unhealthy:" + m.Status.State.String())
				}
			}
			return nil
		}).Should(Succeed())

		By("checking the number of cluster nodes")
		Eventually(func() error {
			return isNodeNumEqual(6)
		}).Should(Succeed())
	})

	It("transition machine state to healthy", func() {
		By("copying dummy redfish data to " + targetIP)
		Eventually(func() error {
			return copyDummyHealthyRedfishDataToWorker(targetIP)
		}).Should(Succeed())

		By("checking machine state")
		Eventually(func() error {
			stdout, _, err := execAt(boot0, "sabactl", "machines", "get", "--ipv4", targetIP)
			if err != nil {
				return err
			}
			var machines []sabakan.Machine
			err = json.Unmarshal(stdout, &machines)
			if err != nil {
				return err
			}
			for _, m := range machines {
				if m.Status.State.String() != "healthy" {
					return errors.New(m.Spec.Serial + " is not healthy:" + m.Status.State.String())
				}
			}
			return nil
		}).Should(Succeed())
	})

	It("removes one extra node which is joined as a result of this test", func() {
		By("removing one extra node")
		_, _, err := execAt(boot0, "neco", "ipmipower", "stop", targetIP)
		Expect(err).ShouldNot(HaveOccurred())

		By("checking the number of cluster nodes")
		Eventually(func() error {
			return isNodeNumEqual(5)
		}).Should(Succeed())

		By("checking the state of the created machine")
		_, _, err = execAt(boot0, "neco", "ipmipower", "start", targetIP)
		Expect(err).ShouldNot(HaveOccurred())

		By("copying dummy redfish data to " + targetIP)
		Eventually(func() error {
			return copyDummyHealthyRedfishDataToWorker(targetIP)
		}).Should(Succeed())

		By("changing mock server response")
		Eventually(func() error {
			stdout, _, err := execAt(boot0, "sabactl", "machines", "get", "--ipv4", targetIP)
			if err != nil {
				return err
			}
			var machines []sabakan.Machine
			err = json.Unmarshal(stdout, &machines)
			if err != nil {
				return err
			}
			for _, m := range machines {
				if m.Status.State.String() != "healthy" {
					return errors.New(m.Spec.Serial + " is not healthy:" + m.Status.State.String())
				}
			}
			return nil
		}).Should(Succeed())

		By("checking the number of cluster nodes")
		Eventually(func() error {
			return isNodeNumEqual(5)
		}).Should(Succeed())
	})
}
