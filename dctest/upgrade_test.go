package dctest

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/cybozu-go/neco"
	sabakan "github.com/cybozu-go/sabakan/client"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func testUpgrade() {
	It("should update neco package", func() {
		By("Changing env for test")
		_, _, err := execAt(boot0, "neco", "config", "set", "env", "test")
		Expect(err).ShouldNot(HaveOccurred())

		By("Wait for systemd unit files to be updated")
		artifacts := []struct {
			service  string
			imageTag string
		}{
			{neco.EtcdService, "quay.io/cybozu/etcd:3.3.9-4"},
			{neco.VaultService, "quay.io/cybozu/vault:0.11.0-3"},
		}
		Eventually(func() error {
			for _, art := range artifacts {
				for _, h := range []string{boot0, boot1, boot2} {
					stdout, _, err := execAt(h, "systemctl", "show", art.service, "--property=ExecStart")
					if err != nil {
						return err
					}
					if !strings.Contains(string(stdout), art.imageTag) {
						return fmt.Errorf("%s is not updated: %s", art.service, string(stdout))
					}
				}
			}
			return nil
		}).Should(Succeed())

		By("Checking status of neco-updater and neco-worker")
		for _, h := range []string{boot0, boot1, boot2} {
			execSafeAt(h, "systemctl", "-q", "is-active", "neco-updater.service")
			execSafeAt(h, "systemctl", "-q", "is-active", "neco-worker.service")
		}

		By("Checking new etcd is running")
		hasNewEtcd := regexp.MustCompile(`etcd\s+quay.io/cybozu/etcd:3.3.9-4\s+running`)
		Eventually(func() error {
			for _, h := range []string{boot0, boot1, boot2} {
				stdout, _, err := execAt(h, "sudo", "rkt", "list")
				if err != nil {
					return err
				}
				if !hasNewEtcd.Match(stdout) {
					return errors.New("etcd is not updated on " + h)
				}
			}
			return nil
		}, 20*time.Minute).Should(Succeed())

		By("Checking newer CoreOS is uploaded")
		Eventually(func() error {
			stdout, stderr, err := execAt(boot0, "sabactl", "images", "index")
			if err != nil {
				return fmt.Errorf("%v: stderr=%s", err, stderr)
			}

			index := new(sabakan.ImageIndex)
			err = json.Unmarshal(stdout, index)
			Expect(err).NotTo(HaveOccurred())

			if index.Find("1911.3.0") == nil {
				return errors.New("index does not contains newer version")
			}
			return nil
		}, 10*time.Minute).Should(Succeed())
	})
}
