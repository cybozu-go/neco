package dctest

import (
	"encoding/json"
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

func testRetireServer() {
	It("retire worker node", func() {
		By("getting all machines")
		allMachines, err := getSabakanMachines()
		Expect(err).NotTo(HaveOccurred())
		Expect(allMachines).NotTo(HaveLen(0))

		By("selecting a target node")
		stdout, stderr, err := execAt(bootServers[0], "kubectl", "get", "nodes", "-l", "cke.cybozu.com/role=ss", "-o", "json")
		Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		var nl corev1.NodeList
		err = json.Unmarshal(stdout, &nl)
		Expect(err).NotTo(HaveOccurred(), "data=%s", stdout)
		targetNodename := nl.Items[0].Name

		By("getting the target node's serial")
		var targetSerial string
		for _, m := range allMachines {
			if m.Spec.IPv4[0] == targetNodename {
				targetSerial = m.Spec.Serial
			}
		}
		Expect(targetSerial).NotTo(BeEmpty())

		By("retiring the target node")
		execSafeAt(bootServers[0], "kubectl", "drain", targetNodename, "--delete-emptydir-data=true", "--ignore-daemonsets=true")
		execSafeAt(bootServers[0], "sabactl", "machines", "set-state", targetSerial, "retiring")

		By("waiting until the node's state becomes retired")
		Eventually(func() error {
			stdout, stderr, err := execAt(bootServers[0], "sabactl", "machines", "get-state", targetSerial)
			if err != nil {
				return fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
			}
			if strings.TrimSpace(string(stdout)) != "retired" {
				return fmt.Errorf("machine state is not retired: %s", string(stdout))
			}
			return nil
		}).Should(Succeed())

		By("confirming that the deletion of disk encryption keys on the sabakan")
		stdout, stderr, err = execEtcdctlAt(bootServers[0], "-w", "json", "get", "/sabakan/crypts/"+targetSerial, "--prefix")
		Expect(err).NotTo(HaveOccurred(), "stderr=%s", stderr)

		var result struct {
			KVS []interface{} `json:"kvs,omitempty"`
		}
		err = json.Unmarshal(stdout, &result)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.KVS).To(HaveLen(0))

		By("waiting until the target node becomes powered off")
		Eventually(func() error {
			stdout, stderr, err := execAt(bootServers[0], "neco", "power", "status", targetSerial)
			if err != nil {
				return fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
			}
			powerStatus := strings.TrimSpace(string(stdout))
			if powerStatus != "Off" {
				return fmt.Errorf("machine power is not off: %s", string(stdout))
			}
			return nil
		}).Should(Succeed())

		By("confirming that orther nodes will not be powered off")
		for _, m := range allMachines {
			if m.Spec.IPv4[0] == targetNodename {
				continue
			}
			if m.Spec.Rack == 3 && m.Spec.Role == "boot" {
				continue
			}
			stdout, stderr, err := execAt(bootServers[0], "neco", "power", "status", m.Spec.Serial)
			Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
			powerStatus := strings.TrimSpace(string(stdout))
			Expect(powerStatus).To(Equal("On"))
		}
	})
}
