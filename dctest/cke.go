package dctest

import (
	"encoding/json"
	"fmt"
	"time"

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

// TestCKESetup tests CKE setup
func TestCKESetup() {
	It("should generates cluster.yml automatically", func() {
		By("setting configurations")
		execSafeAt(boot0, "ckecli", "constraints", "set", "control-plane-count", "3")
		execSafeAt(boot0, "ckecli", "constraints", "set", "minimum-workers", "2")
		execSafeAt(boot0, "ckecli", "sabakan", "set-url", "http://localhost:10080")

		By("waiting for cluster.yml generation")
		Eventually(func() error {
			stdout, stderr, err := execAt(boot0, "ckecli", "cluster", "get")
			if err != nil {
				return fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
			}
			return nil
		}, 20*time.Minute).Should(Succeed())
	})
}

// TestCKE tests CKE
func TestCKE() {
	It("all systemd units are active", func() {
		By("getting systemd unit statuses by serf members")
		Eventually(func() error {
			stdout, stderr, err := execAt(boot0, "serf", "members", "-format", "json", "-tag", "os-name=\"Container Linux by CoreOS\"")
			if err != nil {
				return fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
			}
			var m serfMemberContainer
			err = json.Unmarshal(stdout, &m)
			if err != nil {
				return err
			}
			// Number of worker node is 6
			if len(m.Members) != 6 {
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
			}

			return nil
		}).Should(Succeed())
	})

	It("wait for Kubernetes cluster to become ready", func() {
		By("generating kubeconfig for cluster admin")
		execSafeAt(boot0, "mkdir", "-p", ".kube")
		execSafeAt(boot0, "ckecli", "kubernetes", "issue", ">", ".kube/config")

		By("waiting nodes")
		Eventually(func() error {
			stdout, stderr, err := execAt(boot0, "kubectl", "get", "nodes", "-o", "json")
			if err != nil {
				return fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
			}

			var nl corev1.NodeList
			err = json.Unmarshal(stdout, &nl)
			if err != nil {
				return err
			}

			if len(nl.Items) != 5 {
				return fmt.Errorf("too few nodes: %d", len(nl.Items))
			}
			return nil
		}).Should(Succeed())
	})
}
