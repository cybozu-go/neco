package dctest

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/cybozu-go/neco"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// TestUpgrade test neco debian package upgrade scenario
func TestUpgrade() {
	It("should update neco package", func() {
		etcdContainerImage, err := neco.CurrentArtifacts.FindContainerImage("etcd")
		Expect(err).ShouldNot(HaveOccurred())
		vaultContainerImage, err := neco.CurrentArtifacts.FindContainerImage("vault")
		Expect(err).ShouldNot(HaveOccurred())

		By("Changing env for test")
		_, _, err = execAt(boot0, "neco", "config", "set", "env", "test")
		Expect(err).ShouldNot(HaveOccurred())

		By("Wait for systemd unit files to be updated")
		artifacts := []struct {
			service string
			image   neco.ContainerImage
		}{
			{neco.EtcdService, etcdContainerImage},
			{neco.VaultService, vaultContainerImage},
		}
		Eventually(func() error {
			for _, art := range artifacts {
				for _, h := range []string{boot0, boot1, boot2} {
					stdout, _, err := execAt(h, "systemctl", "show", art.service, "--property=ExecStart")
					if err != nil {
						return err
					}
					if !strings.Contains(string(stdout), art.image.FullName(false)) {
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
		hasNewEtcd := regexp.MustCompile(`etcd\s+` + etcdContainerImage.FullName(false) + `\s+running`)
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
	})
}
