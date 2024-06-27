package dctest

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func testNecoRebooter() {
	It("should switch the leader", func() {
		By("getting leader node name")
		var leaderNodeBefore string
		Eventually(func() error {
			node, err := getNecoRebooterLeaderNode(storage.KeyNecoRebooterLeader)
			if err != nil {
				return err
			}
			leaderNodeBefore = node
			return nil
		}).Should(Succeed())

		By("restarting neco-rebooter on " + leaderNodeBefore)
		index, err := strconv.Atoi(leaderNodeBefore[len(leaderNodeBefore)-1:])
		Expect(err).ShouldNot(HaveOccurred(), "data=%s", leaderNodeBefore[len(leaderNodeBefore)-1:])
		stdout, stderr, err := execAt(bootServers[index], "sudo", "systemctl", "restart", "neco-rebooter.service")
		Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		By("getting leader node name again")
		Eventually(func() error {
			leaderNodeAfter, err := getNecoRebooterLeaderNode(storage.KeyNecoRebooterLeader)
			if err != nil {
				return err
			}

			if leaderNodeAfter == leaderNodeBefore {
				return errors.New("leader is not changed")
			}
			return nil
		}).Should(Succeed())
	})
}

func getNecoRebooterLeaderNode(leaderKeyPrefix string) (string, error) {
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
		log.Info("neco-rebooter: leader key revision of "+string(val), map[string]interface{}{
			"revision": kvs.CreateRevision,
		})

		// revision starts at 1
		// https://github.com/etcd-io/website/blob/master/content/docs/v3.4.0/learning/glossary.md#revision
		if revision == 0 || kvs.CreateRevision < revision {
			revision = kvs.CreateRevision
			value = string(val)
		}
	}
	log.Info("neco-rebooter: leader is "+value, nil)
	return value, nil
}
