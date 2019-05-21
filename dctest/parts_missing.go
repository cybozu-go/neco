package dctest

import (
	"encoding/json"
	"errors"
	"path"

	"github.com/cybozu-go/sabakan/v2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// TestPartsMissing test parts missing scenario
func TestPartsMissing() {
	It("transitition machine state to unhealthy", func() {
		By("changing mock server response")
		fileName := "dummy_redfish_data.json"
		fileContent, err := generateFileContent("OK", "OK", "OK", "PCIeSSD.Slot.XXX", "PCIeSSD.Slot.3-C")
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
	})

	It("transitition machine state to healthy", func() {
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
}
