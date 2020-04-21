package dctest

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/cybozu-go/sabakan/v2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// TestSabakanStateSetter tests the behavior of sabakan-state-setter in bootstrapping
func TestSabakanStateSetter() {
	It("should wait for all nodes to join serf", func() {
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

		By("checking all serf members are active")
		Eventually(func() error {
			m, err := getSerfWorkerMembers()
			if err != nil {
				return err
			}

			// Debug log
			var serfMember []string
			for _, mem := range m.Members {
				serfMember = append(serfMember, mem.Name+":"+mem.Status)
			}
			sort.Strings(serfMember)
			fmt.Printf("%d: %s\n", len(m.Members), strings.Join(serfMember, ","))

			if len(m.Members) != availableNodes {
				return fmt.Errorf("too few serf members. expected %d, actual %d", availableNodes, len(m.Members))
			}

			return nil
		}).Should(Succeed())
	})

	It("should wait for all machines to become healthy", func() {
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

func getMachinesSpecifiedRole(role string) ([]sabakan.Machine, error) {
	stdout, err := func(role string) ([]byte, error) {
		if role == "" {
			stdout, _, err := execAt(bootServers[0], "sabactl", "machines", "get")
			return stdout, err
		}
		stdout, _, err := execAt(bootServers[0], "sabactl", "machines", "get", "--role", role)
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
