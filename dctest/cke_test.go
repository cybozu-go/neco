package dctest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cybozu-go/sabakan/v2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

// serfMember is copied from type Member https://godoc.org/github.com/hashicorp/serf/cmd/serf/command#Member
// to prevent much vendoring
type serfMember struct {
	Name   string            `json:"name"`
	Addr   string            `json:"addr"`
	Port   uint16            `json:"port"`
	Tags   map[string]string `json:"tags"`
	Status string            `json:"status"`
	Proto  map[string]uint8  `json:"protocol"`
	// contains filtered or unexported fields
}

// serfMemberContainer is copied from type MemberContainer https://godoc.org/github.com/hashicorp/serf/cmd/serf/command#MemberContainer
// to prevent much vendoring
type serfMemberContainer struct {
	Members []serfMember `json:"members"`
}

// testCKESetup tests CKE setup
func testCKESetup() {
	It("should generates cluster.yml automatically", func() {
		By("setting configurations")
		execSafeAt(bootServers[0], "ckecli", "constraints", "set", "control-plane-count", "3")
		execSafeAt(bootServers[0], "ckecli", "constraints", "set", "minimum-workers", "2")
		execSafeAt(bootServers[0], "ckecli", "sabakan", "set-url", "http://localhost:10080")

		By("waiting for cluster.yml generation")
		Eventually(func() error {
			stdout, stderr, err := execAt(bootServers[0], "ckecli", "cluster", "get")
			if err != nil {
				return fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
			}
			return nil
		}, 20*time.Minute).Should(Succeed())
	})
}

// testCKE tests CKE
func testCKE() {
	It("all systemd units are active", func() {
		By("getting machines list")
		stdout, _, err := execAt(bootServers[0], "sabactl", "machines", "get", "--role=cs")
		Expect(err).ShouldNot(HaveOccurred())
		var csMachines []sabakan.Machine
		err = json.Unmarshal(stdout, &csMachines)
		Expect(err).ShouldNot(HaveOccurred())

		stdout, _, err = execAt(bootServers[0], "sabactl", "machines", "get", "--role=ss")
		Expect(err).ShouldNot(HaveOccurred())
		var ssMachines []sabakan.Machine
		err = json.Unmarshal(stdout, &ssMachines)
		Expect(err).ShouldNot(HaveOccurred())

		availableNodes := len(csMachines) + len(ssMachines)
		Expect(availableNodes).NotTo(Equal(0))

		By("getting systemd unit statuses by serf members")
		Eventually(func() error {
			m, err := getSerfWorkerMembers()
			if err != nil {
				return err
			}

			if len(m.Members) != availableNodes {
				return fmt.Errorf("too few serf members: %d", len(m.Members))
			}

			for _, member := range m.Members {
				tag, ok := member.Tags["systemd-units-failed"]
				if !ok {
					return fmt.Errorf("member %s does not define tag systemd-units-failed", member.Name)
				}
				if tag != "" {
					return fmt.Errorf("member %s fails systemd units: %s", member.Name, tag)
				}

				serial, ok := member.Tags["serial"]
				if !ok {
					return fmt.Errorf("member %s does not define tag serial", member.Name)
				}
				stdout := execSafeAt(bootServers[0], "sabactl", "machines", "get-state", serial)
				state := string(bytes.TrimSpace(stdout))
				if state != "healthy" {
					return fmt.Errorf("sabakan machine state of member %s is not healthy: %s", member.Name, state)
				}
			}

			return nil
		}).Should(Succeed())
	})

	It("wait for Kubernetes cluster to become ready", func() {
		By("generating kubeconfig for cluster admin")
		execSafeAt(bootServers[0], "mkdir", "-p", ".kube")
		execSafeAt(bootServers[0], "ckecli", "kubernetes", "issue", ">", ".kube/config")

		By("waiting nodes")
		Eventually(func() error {
			stdout, stderr, err := execAt(bootServers[0], "kubectl", "get", "nodes", "-o", "json")
			if err != nil {
				return fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
			}

			var nl corev1.NodeList
			err = json.Unmarshal(stdout, &nl)
			if err != nil {
				return err
			}

			// control-plane-count + minimum-workers = 5
			// https://github.com/cybozu-go/cke/blob/main/docs/sabakan-integration.md#initialization
			if len(nl.Items) != 5 {
				return fmt.Errorf("too few nodes: %d", len(nl.Items))
			}
			return nil
		}).Should(Succeed())
	})
}
