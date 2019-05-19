package dctest

import (
	"encoding/json"
	"errors"

	"github.com/cybozu-go/sabakan/v2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	yaml "gopkg.in/yaml.v2"
)

// TestPartsMissing test parts missing scenario
func TestPartsMissing() {
	var targetIP string

	It("transitition machine state to unhealthy", func() {
		stdout, stderr, err := execAt(boot0, "ckecli",
			"cluster", "get")
		Expect(err).ShouldNot(HaveOccurred(), "stderr=%s", stderr)

		cluster := new(ckeCluster)
		err = yaml.Unmarshal(stdout, cluster)
		Expect(err).ShouldNot(HaveOccurred())

		for _, n := range cluster.Nodes {
			if !n.ControlPlane {
				targetIP = n.Address
				break
			}
		}
		if targetIP == "" {
			err = errors.New("Unable to find non controller node")
		}
		Expect(err).ShouldNot(HaveOccurred())

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
