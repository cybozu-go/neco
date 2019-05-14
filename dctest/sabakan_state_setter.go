package dctest

import (
	"bytes"
	"encoding/json"
	"errors"
	"path"
	"path/filepath"
	"text/template"

	"github.com/cybozu-go/sabakan/v2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// TestSabakanStateSetter tests the bahavior of sabakan-state-setter in bootstrapping
func TestSabakanStateSetter() {
	It("is confirmed that states of all machines are healthy", func() {
		By("copying dummy_redfish_data.json to all nodes")
		fileName := "dummy_redfish_data.json"
		fileContent, err := generateFileContent("OK", "OK", "OK", "PCIeSSD.Slot.2-C", "PCIeSSD.Slot.3-C")
		Expect(err).ShouldNot(HaveOccurred())
		for _, boot := range []string{boot0, boot1, boot2, boot3} {
			_, _, err = execAtWithInput(boot, []byte(fileContent), "dd", "of="+fileName)
			Expect(err).ShouldNot(HaveOccurred())
			_, _, err = execAt(boot, "mv", fileName, filepath.Join("/etc/neco/", fileName))
			Expect(err).ShouldNot(HaveOccurred())
		}
		machines, err := getMachinesSpecifiedRole("worker")
		for _, m := range machines {
			_, _, err = execAt(boot0, "ckecli", "scp", filepath.Join("/etc/neco/", fileName), "cybozu@"+m.Spec.IPv4[0]+":")
			Expect(err).ShouldNot(HaveOccurred())
			_, _, err = execAt(boot0, "ckecli", "ssh", "cybozu@"+m.Spec.IPv4[0], "sudo", "mv", fileName, path.Join("/etc/neco", fileName))
			Expect(err).ShouldNot(HaveOccurred())
		}

		By("checking all machine's state")
		Eventually(func() error {
			machines, err := getMachinesSpecifiedRole("")
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

func generateFileContent(health1, health2, health3, controller1, controller2 string) (string, error) {
	fileContent := `[
    {
        "path": "/redfish/v1/Systems/System.Embedded.1/Processors/CPU.Socket.1",
        "data": {
            "@odata.id": "/redfish/v1/Systems/System.Embedded.1/Processors/CPU.Socket.1",
            "Status": {
                "Health": "OK",
            }
        }
    },
    {
        "path": "/redfish/v1/Systems/System.Embedded.1/Processors/CPU.Socket.2",
        "data": {
            "@odata.id": "/redfish/v1/Systems/System.Embedded.1/Processors/CPU.Socket.2",
            "Status": {
                "Health": "{{ .Health1 }}",
            }
        }
    },
    {
        "path": "/redfish/v1/Systems/System.Embedded.1/Storage/AHCI.Slot.1-1",
        "data": {
            "@odata.id": "/redfish/v1/Systems/System.Embedded.1/Storage/AHCI.Slot.1-1",
            "Status": {
                "Health": "OK",
            }
        }
    },
    {
        "path": "/redfish/v1/Systems/System.Embedded.1/Storage/{{ .Controller1 }}",
        "data": {
            "@odata.id": "/redfish/v1/Systems/System.Embedded.1/Storage/{{ .Controller1 }}",
            "Status": {
                "Health": "{{ .Health2 }}",
            }
        }
    },
    {
        "path": "/redfish/v1/Systems/System.Embedded.1/Storage/{{ .Controller2 }}",
        "data": {
            "@odata.id": "/redfish/v1/Systems/System.Embedded.1/Storage/{{ .Controller2 }}",
            "Status": {
                "Health": "{{ .Health3 }}",
            }
        }
    }
]`

	tmpl := template.Must(template.New("").Parse(fileContent))
	data := struct {
		Health1     string
		Health2     string
		Health3     string
		Controller1 string
		Controller2 string
	}{
		health1,
		health2,
		health3,
		controller1,
		controller2,
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
