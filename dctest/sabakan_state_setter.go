package dctest

import (
	"encoding/json"
	"errors"
	"path/filepath"

	"github.com/cybozu-go/sabakan/v2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const dummyRedfishDataFile = "dummy_redfish_data.json"
const prefix = "/redfish/v1/Systems/System.Embedded.1/"

// TestSabakanStateSetter tests the bahavior of sabakan-state-setter in bootstrapping
func TestSabakanStateSetter() {
	It("is confirmed that states of all machines are healthy", func() {
		By("copying all healthy dummy redfish data")
		machines, err := getMachinesSpecifiedRole("")
		Expect(err).ShouldNot(HaveOccurred())
		state := map[string]string{
			prefix + "Processors/CPU.Socket.1":  "OK",
			prefix + "Processors/CPU.Socket.2":  "OK",
			prefix + "Storage/AHCI.Slot.1-1":    "OK",
			prefix + "Storage/PCIeSSD.Slot.2-C": "OK",
			prefix + "Storage/PCIeSSD.Slot.3-C": "OK",
			prefix + "Storage/SATAHDD.Slot.1":   "OK",
			prefix + "Storage/SATAHDD.Slot.2":   "OK",
		}
		for _, m := range machines {
			err = copyDummyRedfishDataToWorker(m.Spec.IPv4[0], generateRedfishDummyData(state))
			Expect(err).ShouldNot(HaveOccurred())
		}

		By("checking all machine's state")
		Eventually(func() error {
			machines, err := getMachinesSpecifiedRole("")
			if err != nil {
				return err
			}
			for _, m := range machines {
				if m.Spec.Rack == 3 && m.Spec.Role == "boot" {
					continue
				}
				if m.Status.State.String() != "healthy" {
					return errors.New(m.Spec.Serial + " is not healthy:" + m.Status.State.String())
				}
			}
			return nil
		}).Should(Succeed())
	})
}

func generateRedfishDummyData(data map[string]string) string {
	var result []interface{}

	for k, v := range data {
		entry := map[string]interface{}{
			"path": k,
			"data": map[string]interface{}{
				"Status": map[string]string{
					"Health": v,
				},
			},
		}
		result = append(result, entry)
	}

	json, err := json.Marshal(result)
	Expect(err).ShouldNot(HaveOccurred())
	return string(json)
}

func getMachinesSpecifiedRole(role string) ([]sabakan.Machine, error) {
	stdout, err := func(role string) ([]byte, error) {
		if role == "" {
			stdout, _, err := execAt(boot0, "sabactl", "machines", "get")
			return stdout, err
		}
		stdout, _, err := execAt(boot0, "sabactl", "machines", "get", "--role", role)
		return stdout, err
	}(role)

	if err != nil {
		return nil, err
	}
	var machines []sabakan.Machine
	err = json.Unmarshal(stdout, &machines)
	if err != nil {
		return nil, err
	}
	return machines, nil
}

func copyDummyCPUWarningRedfishDataToWorker(ip string) error {
	state := map[string]string{
		prefix + "Processors/CPU.Socket.1":  "OK",
		prefix + "Processors/CPU.Socket.2":  "Warning",
		prefix + "Storage/AHCI.Slot.1-1":    "OK",
		prefix + "Storage/PCIeSSD.Slot.2-C": "OK",
		prefix + "Storage/PCIeSSD.Slot.3-C": "OK",
		prefix + "Storage/SATAHDD.Slot.1":   "OK",
		prefix + "Storage/SATAHDD.Slot.2":   "OK",
	}

	return copyDummyRedfishDataToWorker(ip, generateRedfishDummyData(state))
}

func copyDummyHDDWarningSingleRedfishDataToWorker(ip string) error {
	state := map[string]string{
		prefix + "Processors/CPU.Socket.1":  "OK",
		prefix + "Processors/CPU.Socket.2":  "OK",
		prefix + "Storage/AHCI.Slot.1-1":    "OK",
		prefix + "Storage/PCIeSSD.Slot.2-C": "OK",
		prefix + "Storage/PCIeSSD.Slot.3-C": "OK",
		prefix + "Storage/SATAHDD.Slot.1":   "OK",
		prefix + "Storage/SATAHDD.Slot.2":   "Warning",
	}

	return copyDummyRedfishDataToWorker(ip, generateRedfishDummyData(state))
}

func copyDummyHDDWarningAllRedfishDataToWorker(ip string) error {
	const prefix = "/redfish/v1/Systems/System.Embedded.1/"
	state := map[string]string{
		prefix + "Processors/CPU.Socket.1":  "OK",
		prefix + "Processors/CPU.Socket.2":  "OK",
		prefix + "Storage/AHCI.Slot.1-1":    "OK",
		prefix + "Storage/PCIeSSD.Slot.2-C": "OK",
		prefix + "Storage/PCIeSSD.Slot.3-C": "OK",
		prefix + "Storage/SATAHDD.Slot.1":   "Warning",
		prefix + "Storage/SATAHDD.Slot.2":   "Warning",
	}

	return copyDummyRedfishDataToWorker(ip, generateRedfishDummyData(state))
}

func copyDummyRedfishDataToWorker(ip, json string) error {
	_, _, err := execAtWithInput(boot0, []byte(json), "dd", "of="+dummyRedfishDataFile)
	if err != nil {
		return err
	}
	_, _, err = execAt(boot0, "ckecli", "scp", dummyRedfishDataFile, "cybozu@"+ip+":")
	if err != nil {
		return err
	}
	_, _, err = execAt(boot0, "ckecli", "ssh", "cybozu@"+ip, "sudo", "mv", dummyRedfishDataFile, filepath.Join("/etc/neco", dummyRedfishDataFile))
	return err
}

func deleteDummyRedfishDataFromWorker(ip string) error {
	_, _, err := execAt(boot0, "ckecli", "ssh", "cybozu@"+ip, "sudo", "rm", filepath.Join("/etc/neco", dummyRedfishDataFile))
	return err
}
