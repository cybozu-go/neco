package dctest

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// testSabakanStateSetter tests the behavior of sabakan-state-setter in bootstrapping
func testSabakanStateSetter() {
	It("should wait for all nodes to join serf", func() {
		By("getting machines list")
		machines, err := getSabakanMachines("--without-role=boot")
		Expect(err).NotTo(HaveOccurred())
		availableNodeCount := len(machines)
		Expect(availableNodeCount).NotTo(Equal(0))

		By("checking all serf members are active")
		Eventually(func() error {
			m, err := getSerfWorkerMembers()
			if err != nil {
				return err
			}

			if len(m.Members) != availableNodeCount {
				return fmt.Errorf("too few serf members. expected %d, actual %d", availableNodeCount, len(m.Members))
			}

			return nil
		}).Should(Succeed())
	})

	It("should wait for all machines to become healthy", func() {
		Eventually(func() error {
			machines, err := getSabakanMachines()
			if err != nil {
				return err
			}
			for _, m := range machines {
				if m.Spec.Rack >= 3 && m.Spec.Role == "boot" {
					continue
				}
				if m.Status.State.String() != "healthy" {
					return errors.New(m.Spec.Serial + " is not healthy:" + m.Status.State.String())
				}
			}
			return nil
		}).Should(Succeed())
	})

	It("should switch the leader", func() {
		By("getting leader node name")
		var leaderNodeBefore string
		Eventually(func() error {
			node, err := getLeaderNode(storage.KeySabakanStateSetterLeader)
			if err != nil {
				return err
			}
			leaderNodeBefore = node
			return nil
		}).Should(Succeed())

		By("restarting sabakan-state-setter on " + leaderNodeBefore)
		index, err := strconv.Atoi(leaderNodeBefore[len(leaderNodeBefore)-1:])
		Expect(err).ShouldNot(HaveOccurred(), "data=%s", leaderNodeBefore[len(leaderNodeBefore)-1:])
		stdout, stderr, err := execAt(bootServers[index], "sudo", "systemctl", "restart", "sabakan-state-setter.service")
		Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		By("getting leader node name again")
		Eventually(func() error {
			leaderNodeAfter, err := getLeaderNode(storage.KeySabakanStateSetterLeader)
			if err != nil {
				return err
			}

			if leaderNodeAfter == leaderNodeBefore {
				return errors.New("leader is not changed")
			}
			return nil
		}).Should(Succeed())
	})

	It("should change shutdown job schedule", func() {
		// Run the shutdown job every minute in dctest.
		for _, boot := range bootServers {
			execSafeAt(boot, "sudo", "sed", "-i", "'s/shutdown-schedule:.*/shutdown-schedule: \"@every 1m\"/'", "/usr/share/neco/sabakan-state-setter.yml")
			execSafeAt(boot, "sudo", "systemctl", "restart", "sabakan-state-setter.service")
		}

		By("checking status of sabakan-state-setter")
		Eventually(func() error {
			for _, boot := range bootServers {
				stdout, _, err := execAt(boot, "systemctl", "is-active", "sabakan-state-setter.service")
				if err != nil {
					return fmt.Errorf("sabakan-state-setter on %s is not active: %s", boot, stdout)
				}
			}
			return nil
		}).Should(Succeed())
	})
}

func getLeaderNode(leaderKeyPrefix string) (string, error) {
	stdout, _, err := execEtcdctlAt(bootServers[0], "-w", "json", "get", neco.NecoPrefix+leaderKeyPrefix, "--prefix")
	if err != nil {
		return "", err
	}

	var result struct {
		KVS []*struct {
			CreateRevision int    `json:"create_revision"`
			Value          string `json:"value"`
		} `json:"kvs"`
	}
	err = json.Unmarshal(stdout, &result)
	if err != nil {
		return "", err
	}
	if len(result.KVS) == 0 {
		return "", errors.New("there is no candidate")
	}

	var revision int
	var value string
	for _, kvs := range result.KVS {
		val, err := base64.StdEncoding.DecodeString(kvs.Value)
		if err != nil {
			return "", err
		}
		log.Info("sabakan-state-setter: leader key revision of "+string(val), map[string]interface{}{
			"revision": kvs.CreateRevision,
		})

		// revision starts at 1
		// https://github.com/etcd-io/website/blob/master/content/docs/v3.4.0/learning/glossary.md#revision
		if revision == 0 || kvs.CreateRevision < revision {
			revision = kvs.CreateRevision
			value = string(val)
		}
	}
	log.Info("sabakan-state-setter: leader is "+value, nil)
	return value, nil
}
