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

// TestPartsFailure test parts failure scenario
func TestPartsFailure() {
	It("transition machine state to unhealthy", func() {
		By("changing mock server response")
		fileName := "dummy_redfish_data.json"
		fileContent, err := generateFileContent("Warning", "OK", "OK", "PCIeSSD.Slot.2-C", "PCIeSSD.Slot.3-C")
		Expect(err).ShouldNot(HaveOccurred())
		_, _, err = execAtWithInput(boot0, []byte(fileContent), "dd", "of="+fileName)
		Expect(err).ShouldNot(HaveOccurred())
		_, _, err = execAt(boot0, "ckecli", "scp", fileName, "cybozu@10.69.0.5:")
		Expect(err).ShouldNot(HaveOccurred())
		_, _, err = execAt(boot0, "ckecli", "ssh", "cybozu@10.69.0.5", "sudo", "mv", fileName, path.Join("/etc/neco", fileName))
		Expect(err).ShouldNot(HaveOccurred())

		By("checking machine state")
		Eventually(func() error {
			stdout, _, err := execAt(boot0, "sabactl", "machines", "get", "--ipv4", "10.69.0.5")
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
			stdout, _, err := execAt(boot0, "kubectl", "get", "nodes", "-o", "json")
			if err != nil {
				return err
			}
			var nl corev1.NodeList
			err = json.Unmarshal(stdout, &nl)
			if err != nil {
				return err
			}
			if len(nl.Items) != 6 {
				return fmt.Errorf("cluster node should be 6, but %d", len(nl.Items))
			}
			return nil
		}).Should(Succeed())
	})

	It("transition machine state to healthy", func() {
		By("changing mock server response")
		fileName := "dummy_redfish_data.json"
		fileContent, err := generateFileContent("OK", "OK", "OK", "PCIeSSD.Slot.2-C", "PCIeSSD.Slot.3-C")
		Expect(err).ShouldNot(HaveOccurred())
		_, _, err = execAtWithInput(boot0, []byte(fileContent), "dd", "of="+fileName)
		Expect(err).ShouldNot(HaveOccurred())
		_, _, err = execAt(boot0, "ckecli", "scp", fileName, "cybozu@10.69.0.5:")
		Expect(err).ShouldNot(HaveOccurred())
		_, _, err = execAt(boot0, "ckecli", "ssh", "cybozu@10.69.0.5", "sudo", "mv", fileName, path.Join("/etc/neco", fileName))
		Expect(err).ShouldNot(HaveOccurred())

		By("checking machine state")
		Eventually(func() error {
			stdout, _, err := execAt(boot0, "sabactl", "machines", "get", "--ipv4", "10.69.0.5")
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
		stdout, _, err := execAt(boot0, "sabactl", "machines", "get", "--role", "worker")
		Expect(err).ShouldNot(HaveOccurred())
		var machines []sabakan.Machine
		err = json.Unmarshal(stdout, &machines)
		Expect(err).ShouldNot(HaveOccurred())
		targetIP := machines[0].Spec.IPv4[0]
		stdout, _, err = execAt(boot0, "neco", "ipmipower", "stop", targetIP)
		Eventually(func() error {
			return nil
		}).Should(Succeed())
	})
}
