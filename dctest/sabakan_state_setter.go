package dctest

import (
	"bytes"
	"encoding/json"
	"errors"
	"path/filepath"
	"text/template"

	"github.com/cybozu-go/sabakan/v2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const dummyRedfishDataFile = "dummy_redfish_data.json"

// TestSabakanStateSetter tests the bahavior of sabakan-state-setter in bootstrapping
func TestSabakanStateSetter() {
	It("is confirmed that states of all machines are healthy", func() {
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

func generateFileContent(health1, health2, health3, health4, health5, controller1, controller2, device1, device2 string) (string, error) {
	fileContent := `[
    {
        "path": "/redfish/v1/Systems/System.Embedded.1/Processors/CPU.Socket.1",
        "data": {
            "Status": {
                "Health": "OK"
            }
        }
    },
    {
        "path": "/redfish/v1/Systems/System.Embedded.1/Processors/CPU.Socket.2",
        "data": {
            "Status": {
                "Health": "{{ .Health1 }}"
            }
        }
    },
    {
        "path": "/redfish/v1/Systems/System.Embedded.1/Storage/AHCI.Slot.1-1",
        "data": {
            "Status": {
                "Health": "OK"
            }
        }
    },
    {
        "path": "/redfish/v1/Systems/System.Embedded.1/Storage/{{ .Controller1 }}",
        "data": {
            "Status": {
                "Health": "{{ .Health2 }}"
            }
        }
    },
    {
        "path": "/redfish/v1/Systems/System.Embedded.1/Storage/{{ .Controller2 }}",
        "data": {
            "Status": {
                "Health": "{{ .Health3 }}"
            }
        }
    },
    {
        "path": "/redfish/v1/Systems/System.Embedded.1/Storage/{{ .Device1 }}",
        "data": {
            "Status": {
                "Health": "{{ .Health4 }}"
            }
        }
    },
    {
        "path": "/redfish/v1/Systems/System.Embedded.1/Storage/{{ .Device2 }}",
        "data": {
            "Status": {
                "Health": "{{ .Health5 }}"
            }
        }
    }
]`

	tmpl := template.Must(template.New("").Parse(fileContent))
	data := struct {
		Health1     string
		Health2     string
		Health3     string
		Health4     string
		Health5     string
		Controller1 string
		Controller2 string
		Device1     string
		Device2     string
	}{
		health1,
		health2,
		health3,
		health4,
		health5,
		controller1,
		controller2,
		device1,
		device2,
	}
	buff := new(bytes.Buffer)
	err := tmpl.Execute(buff, data)
	if err != nil {
		return "", err
	}
	return buff.String(), nil
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
	return copyDummyRedfishDataToWorker(ip, "Warning", "OK", "OK", "OK", "OK", "PCIeSSD.Slot.2-C", "PCIeSSD.Slot.3-C", "SATAHDD.Slot.1", "SATAHDD.SLot.2")
}

func copyDummyHDDWarningSingleRedfishDataToWorker(ip string) error {
	return copyDummyRedfishDataToWorker(ip, "OK", "OK", "OK", "OK", "Warning", "PCIeSSD.Slot.2-C", "PCIeSSD.Slot.3-C", "SATAHDD.Slot.1", "SATAHDD.SLot.2")
}

func copyDummyHDDWarningAllRedfishDataToWorker(ip string) error {
	return copyDummyRedfishDataToWorker(ip, "OK", "OK", "OK", "Warning", "Warning", "PCIeSSD.Slot.2-C", "PCIeSSD.Slot.3-C", "SATAHDD.Slot.1", "SATAHDD.SLot.2")
}

func copyDummyRedfishDataToWorker(ip, health1, health2, health3, health4, health5, controller1, controller2, device1, device2 string) error {
	fileContent, err := generateFileContent(health1, health2, health3, health4, health5, controller1, controller2, device1, device2)
	if err != nil {
		return err
	}
	_, _, err = execAtWithInput(boot0, []byte(fileContent), "dd", "of="+dummyRedfishDataFile)
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
