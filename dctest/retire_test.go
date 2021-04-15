package dctest

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cybozu-go/sabakan/v2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

func testRetireServer() {
	It("retire worker node", func() {
		By("getting a target node name")
		stdout, stderr, err := execAt(bootServers[0], "kubectl", "get", "nodes", "-l", "cke.cybozu.com/role=ss", "-o", "json")
		Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		var nl corev1.NodeList
		err = json.Unmarshal(stdout, &nl)
		Expect(err).NotTo(HaveOccurred())
		nodename := nl.Items[0].Name

		By("getting the serial")
		stdout, stderr, err = execAt(bootServers[0], "sabactl", "machines", "get", "--ipv4", nodename)
		Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		var machines []sabakan.Machine
		err = json.Unmarshal(stdout, &machines)
		Expect(err).NotTo(HaveOccurred())
		Expect(machines).To(HaveLen(1))
		serial := machines[0].Spec.Serial

		By("retiring the node")
		execSafeAt(bootServers[0], "kubectl", "drain", nodename, "--delete-local-data=true", "--ignore-daemonsets=true")
		execSafeAt(bootServers[0], "sabactl", "machines", "set-state", serial, "retiring")

		By("waiting until the node's state becomes retired")
		Eventually(func() error {
			stdout, stderr, err := execAt(bootServers[0], "sabactl", "machines", "get-state", serial)
			if err != nil {
				return fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
			}
			if strings.TrimSpace(string(stdout)) != "retired" {
				return fmt.Errorf("machine state is not retired: %s", string(stdout))
			}
			return nil
		}).Should(Succeed())

		By("confirming the deletion of disk encryption keys on the sabakan")
		stdout, stderr, err = execEtcdctlAt(bootServers[0], "-w", "json", "get", "/sabakan/crypts/"+serial, "--prefix")
		Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		var result struct {
			KVS []interface{} `json:"kvs,omitempty"`
		}
		err = json.Unmarshal(stdout, &result)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.KVS).To(HaveLen(0))
	})
}
