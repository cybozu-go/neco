package dctest

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/cybozu-go/sabakan/v2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// TestSabakanStateSetter tests the behavior of sabakan-state-setter in bootstrapping
func TestSabakanStateSetter(availableNodes int) {
	It("should wait for all nodes to join serf", func() {
		By("checking all serf members are active")
		Eventually(func() error {
			m, err := getSerfWorkerMembers()
			if err != nil {
				return err
			}

			if len(m.Members) != availableNodes {
				return fmt.Errorf("too few serf members: %d", len(m.Members))
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
