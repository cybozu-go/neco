package dctest

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/cybozu-go/sabakan/v2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	yaml "gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
)

const prefix = "/redfish/v1/Systems/System.Embedded.1/"

func isNodeNumEqual(num int) error {
	stdout, stderr, err := execAt(boot0, "kubectl", "get", "nodes", "-o", "json")
	if err != nil {
		return fmt.Errorf("kubectl get nodes -o json failed. err: %v, stdout: %s, stderr: %s", err, stdout, stderr)
	}
	var nl corev1.NodeList
	err = json.Unmarshal(stdout, &nl)
	if err != nil {
		return fmt.Errorf("unmarshal failed. err: %v", err)
	}
	if len(nl.Items) != num {
		return fmt.Errorf("cluster node should be %d, but %d", num, len(nl.Items))
	}
	return nil
}

// TestPartsFailure test parts failure scenario
func TestPartsFailure() {
	It("transition machine state to unhealthy due to cpu warning", func() {
		targetIP := getTargetNode()

		By("copying dummy redfish data with cpu warning to " + targetIP)
		state := map[string]string{
			prefix + "Processors/CPU.Socket.1":  "OK",
			prefix + "Processors/CPU.Socket.2":  "Warning",
			prefix + "Storage/AHCI.Slot.1-1":    "OK",
			prefix + "Storage/PCIeSSD.Slot.2-C": "OK",
			prefix + "Storage/PCIeSSD.Slot.3-C": "OK",
			prefix + "Storage/SATAHDD.Slot.1":   "OK",
			prefix + "Storage/SATAHDD.Slot.2":   "OK",
		}
		data := generateRedfishDummyData(state)
		Eventually(func() error {
			return copyDummyRedfishDataToWorker(targetIP, data)
		}).Should(Succeed())

		By("checking machine state")
		Eventually(func() error {
			return checkHealthOfTarget(targetIP, "unhealthy")
		}).Should(Succeed())

		By("checking the number of cluster nodes")
		Eventually(func() error {
			return isNodeNumEqual(6)
		}).Should(Succeed())
	})

	It("transition machine state to unhealthy due to warning disks become larger than one", func() {
		targetIP := getTargetNode()

		By("copying dummy redfish data with two disk warnings to " + targetIP)
		state := map[string]string{
			prefix + "Processors/CPU.Socket.1":  "OK",
			prefix + "Processors/CPU.Socket.2":  "OK",
			prefix + "Storage/AHCI.Slot.1-1":    "OK",
			prefix + "Storage/PCIeSSD.Slot.2-C": "OK",
			prefix + "Storage/PCIeSSD.Slot.3-C": "OK",
			prefix + "Storage/SATAHDD.Slot.1":   "Warning",
			prefix + "Storage/SATAHDD.Slot.2":   "Warning",
		}
		data := generateRedfishDummyData(state)
		Eventually(func() error {
			return copyDummyRedfishDataToWorker(targetIP, data)
		}).Should(Succeed())

		By("checking machine state")
		Eventually(func() error {
			return checkHealthOfTarget(targetIP, "unhealthy")
		}).Should(Succeed())

		By("checking the number of cluster nodes")
		Eventually(func() error {
			return isNodeNumEqual(6)
		}).Should(Succeed())
	})

	It("transition machine state to healthy even one disk warning occurred", func() {
		targetIP := getTargetNode()

		By("copying dummy data with one disk warning to " + targetIP)
		state := map[string]string{
			prefix + "Processors/CPU.Socket.1":  "OK",
			prefix + "Processors/CPU.Socket.2":  "OK",
			prefix + "Storage/AHCI.Slot.1-1":    "OK",
			prefix + "Storage/PCIeSSD.Slot.2-C": "OK",
			prefix + "Storage/PCIeSSD.Slot.3-C": "OK",
			prefix + "Storage/SATAHDD.Slot.1":   "OK",
			prefix + "Storage/SATAHDD.Slot.2":   "Warning",
		}
		data := generateRedfishDummyData(state)
		Eventually(func() error {
			return copyDummyRedfishDataToWorker(targetIP, data)
		}).Should(Succeed())

		By("checking machine state")
		Eventually(func() error {
			return checkHealthOfTarget(targetIP, "healthy")
		}).Should(Succeed())

		By("checking the number of cluster nodes")
		Eventually(func() error {
			return isNodeNumEqual(6)
		}).Should(Succeed())
	})
}

func getTargetNode() string {
	stdout, stderr, err := execAt(boot0, "ckecli", "cluster", "get")
	Expect(err).ShouldNot(HaveOccurred(), "stderr=%s", stderr)

	cluster := new(ckeCluster)
	err = yaml.Unmarshal(stdout, cluster)
	Expect(err).ShouldNot(HaveOccurred())

	var targetIP string
	for _, n := range cluster.Nodes {
		if !n.ControlPlane {
			targetIP = n.Address
			break
		}
	}
	Expect(targetIP).NotTo(Equal(""))

	return targetIP
}

func checkHealthOfTarget(targetIP string, health string) error {
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
		if m.Status.State.String() != health {
			return errors.New(m.Spec.Serial + " is not unhealthy:" + m.Status.State.String())
		}
	}
	return nil
}
