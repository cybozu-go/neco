package dctest

import (
	"encoding/json"
	"errors"
	"fmt"
	"path"

	"github.com/cybozu-go/sabakan/v2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

func copyDummyRedfishData(ip, health1, health2, health3 string) error {
	fileName := "dummy_redfish_data.json"
	fileContent, err := generateFileContent(health1, health2, health3, "PCIeSSD.Slot.2-C", "PCIeSSD.Slot.3-C")
	if err != nil {
		return err
	}
	_, _, err = execAtWithInput(boot0, []byte(fileContent), "dd", "of="+fileName)
	if err != nil {
		return err
	}

	_, _, err = execAt(boot0, "ckecli", "scp", fileName, "cybozu@"+ip+":")
	if err != nil {
		return err
	}
	_, _, err = execAt(boot0, "ckecli", "ssh", "cybozu@"+ip, "sudo", "mv", fileName, path.Join("/etc/neco", fileName))
	if err != nil {
		return err
	}
	return nil
}

func isNodeNumEqual(num int) error {
	stdout, _, err := execAt(boot0, "kubectl", "get", "nodes", "-o", "json")
	if err != nil {
		return err
	}
	var nl corev1.NodeList
	err = json.Unmarshal(stdout, &nl)
	if err != nil {
		return err
	}
	if len(nl.Items) != num {
		return fmt.Errorf("cluster node should be %d, but %d", num, len(nl.Items))
	}
	return nil
}

// TestPartsFailure test parts failure scenario
func TestPartsFailure() {
	var targetIP string

	It("transition machine state to unhealthy", func() {
		stdout, _, err := execAt(boot0, "kubectl", "get", "nodes", "-o", "json")
		Expect(err).ShouldNot(HaveOccurred())

		var nl corev1.NodeList
		err = json.Unmarshal(stdout, &nl)
		Expect(err).ShouldNot(HaveOccurred())
		targetIP = nl.Items[0].Name

		By("copying dummy redfish data to" + targetIP)
		Eventually(func() error {
			return copyDummyRedfishData(targetIP, "Warning", "OK", "OK")
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
		By("copying dummy redfish data to" + targetIP)
		Eventually(func() error {
			return copyDummyRedfishData(targetIP, "OK", "OK", "OK")
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

	It("removes one extra node which is joined as a result of unhealthy machine", func() {
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

		By("copying dummy redfish data to" + targetIP)
		Eventually(func() error {
			return copyDummyRedfishData(targetIP, "OK", "OK", "OK")
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
