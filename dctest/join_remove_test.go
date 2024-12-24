package dctest

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// testJoinRemove test boot server join/remove scenario
func testJoinRemove() {
	It("copies root CA certificate from existing server", func() {
		stdout, stderr, err := execAt(bootServers[0], "cat", neco.ServerCAFile)
		Expect(err).ShouldNot(HaveOccurred(), "stderr=%s", stderr)
		_, stderr, err = execAtWithInput(allBootServers[3], stdout, "sudo", "tee", neco.ServerCAFile)
		Expect(err).ShouldNot(HaveOccurred(), "stderr=%s", stderr)
		stdout, stderr, err = execAt(allBootServers[3], "sudo", "update-ca-certificates")
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
	})

	It("should add a new boot server", func() {
		// for upgrading test; install the same version of Neco as upgraded boot servers.
		remoteFilename := filepath.Join("/tmp", filepath.Base(debFile))
		stdout, stderr, err := execAt(allBootServers[3], "sudo", "dpkg", "-i", remoteFilename)
		Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		token := getVaultToken()
		stdout, stderr, err = execAt(
			allBootServers[3], "sudo", "env", "VAULT_TOKEN="+token, "neco", "join", "0", "1", "2")
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		execSafeAt(allBootServers[3], "test", "-f", neco.NecoConfFile)
		execSafeAt(allBootServers[3], "test", "-f", neco.NecoCertFile)
		execSafeAt(allBootServers[3], "test", "-f", neco.NecoKeyFile)

		execSafeAt(allBootServers[3], "test", "-f", neco.EtcdBackupCertFile)
		execSafeAt(allBootServers[3], "test", "-f", neco.EtcdBackupKeyFile)
		execSafeAt(allBootServers[3], "test", "-f", neco.TimerFile("etcd-backup"))
		execSafeAt(allBootServers[3], "test", "-f", neco.ServiceFile("etcd-backup"))

		execSafeAt(allBootServers[3], "test", "-f", "/lib/systemd/system/neco-updater.service")
		execSafeAt(allBootServers[3], "test", "-f", "/lib/systemd/system/neco-worker.service")
		execSafeAt(allBootServers[3], "test", "-f", "/lib/systemd/system/neco-rebooter.service")
		execSafeAt(allBootServers[3], "test", "-f", "/lib/systemd/system/node-exporter.service")
		execSafeAt(allBootServers[3], "test", "-f", "/lib/systemd/system/sabakan-state-setter.service")

		execSafeAt(allBootServers[3], "systemctl", "-q", "is-active", "neco-updater.service")
		execSafeAt(allBootServers[3], "systemctl", "-q", "is-active", "neco-worker.service")
		execSafeAt(allBootServers[3], "systemctl", "-q", "is-active", "node-exporter.service")
		execSafeAt(allBootServers[3], "systemctl", "-q", "is-active", "etcd-backup.timer")
	})

	It("should install programs", func() {
		By("Waiting for request to complete")
		waitRequestComplete("members: [0 1 2 3]")

		By("Waiting for etcd to be restarted on boot-0")
		time.Sleep(time.Second * 7)

		By("Checking etcd installation")
		stdout, stderr, err := execAt(allBootServers[3], "systemctl", "-q", "is-active", neco.EtcdService+".service")
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
		stdout, stderr, err = execAt(allBootServers[3], "test", "-f", "/usr/local/bin/etcdctl")
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
		By("Checking vault installation")
		stdout, stderr, err = execAt(allBootServers[3], "systemctl", "-q", "is-active", neco.VaultService+".service")
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
	})

	It("should setup sabakan on boot-3", func() {
		stdout, stderr, err := execAt(
			allBootServers[3], "sudo", "env", "VAULT_TOKEN="+getVaultToken(), "neco", "init-local", "sabakan")
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		execSafeAt(allBootServers[3], "test", "-d", neco.SabakanDataDir)
		execSafeAt(allBootServers[3], "test", "-f", neco.SabakanConfFile)
		execSafeAt(allBootServers[3], "test", "-f", neco.SabakanEtcdKeyFile)
		execSafeAt(allBootServers[3], "test", "-f", neco.SabakanEtcdCertFile)
		execSafeAt(allBootServers[3], "test", "-f", neco.SabactlBashCompletionFile)

		execSafeAt(allBootServers[3], "systemctl", "-q", "is-active", "sabakan.service")
		execSafeAt(allBootServers[3], "systemctl", "-q", "is-active", "sabakan-state-setter.service")
	})

	It("should setup cke on boot-3", func() {
		token := getVaultToken()

		stdout, stderr, err := execAt(
			allBootServers[3], "sudo", "env", "VAULT_TOKEN="+token, "neco", "init-local", "cke")
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		execSafeAt(allBootServers[3], "test", "-f", neco.CKEConfFile)
		execSafeAt(allBootServers[3], "test", "-f", neco.CKEKeyFile)
		execSafeAt(allBootServers[3], "test", "-f", neco.CKECertFile)

		execSafeAt(allBootServers[3], "systemctl", "-q", "is-active", "cke.service")
	})

	It("should run neco-rebooter on boot-3", func() {
		Eventually(func(g Gomega) {
			stdout, stderr, err := execAt(allBootServers[3], "test", "-f", neco.NecoRebooterConfFile)
			g.Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
			stdout, stderr, err = execAt(allBootServers[3], "systemctl", "-q", "is-active", "neco-rebooter.service")
			g.Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		}).Should(Succeed())
	})

	It("should setup boot-3 hardware", func() {
		Eventually(func() error {
			stdout, stderr, err := execAt(allBootServers[3], "sudo", "neco", "bmc", "setup-hw")
			if err != nil {
				return fmt.Errorf("neco bmc setup-hw failed; host: %s, err: %s, stdout: %s, stderr: %s", allBootServers[3], err, stdout, stderr)
			}
			return nil
		}).Should(Succeed())
	})

	It("should add boot-3 to etcd cluster", func() {
		stdout, stderr, err := execEtcdctlAt(bootServers[0], "-w", "json", "member", "list")
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
		var mlr struct {
			Members []struct {
				Name string `json:"name"`
			} `json:"members"`
		}

		err = json.Unmarshal(stdout, &mlr)
		Expect(err).ShouldNot(HaveOccurred(), "data=%s", stdout)

		names := make([]string, len(mlr.Members))
		for i, m := range mlr.Members {
			names[i] = m.Name
		}
		Expect(names).Should(ContainElement("boot-3"))
	})

	It("should reset failed status of systemd-networkd-wait-online.service on boot-3", func() {
		// systemd-networkd-wait-online.service may fail due to timeout on dctest env
		// Frankly speaking, it is not a problem because the test program can already connect to boot servers.
		execSafeAt(allBootServers[3], "sudo", "systemctl", "reset-failed", "systemd-networkd-wait-online.service")
		// etcd-backup.service may fail due to timeout on dctest env
		// etcdctl is installed by worker, so it may not be ready when etcd-backup.service is started.
		execSafeAt(allBootServers[3], "sudo", "systemctl", "reset-failed", "etcd-backup.service")
	})

	It("should set state of boot-3 to healthy", func() {
		By("Checking boot-3 machine state")
		Eventually(func() error {
			serial := fmt.Sprintf("%x", sha1.Sum([]byte("boot-3")))
			stdout := execSafeAt(bootServers[0], "sabactl", "machines", "get-state", serial)
			state := string(bytes.TrimSpace(stdout))
			if state != "healthy" {
				return errors.New("boot-3 machine state is not healthy: " + state)
			}
			return nil
		}).Should(Succeed())
	})

	It("should remove boot-3", func() {
		By("Running neco leave 3")
		token := getVaultToken()
		execSafeAt(bootServers[0], "sudo", "env", "VAULT_TOKEN="+token, "neco", "leave", "3")

		By("Waiting for the request to complete")
		waitRequestComplete("members: [0 1 2]", true)

		By("Waiting boot-3 gets removed from etcd")
		Eventually(func() error {
			stdout, _, err := execEtcdctlAt(bootServers[0], "-w", "json", "member", "list")
			if err != nil {
				return err
			}

			var mlr struct {
				Members []struct {
					Name string `json:"name"`
				} `json:"members"`
			}
			err = json.Unmarshal(stdout, &mlr)
			if err != nil {
				return err
			}

			for _, m := range mlr.Members {
				if m.Name == "boot-3" {
					return errors.New("boot-3 is not removed from etcd")
				}
			}
			return nil
		}).Should(Succeed())

		// need to wait for boot-1/2 to restart etcd, or the system would become
		// unstable during tests.
		time.Sleep(3 * time.Minute)
	})

	It("should set state of boot-3 to unreachable", func() {
		By("Stopping boot-3")
		// In DCtest on CircleCI, ginkgo is executed in the operation pod, so you cannot use pmctl in this context.
		// This error is ignored deliberately, because this SSH session is closed by remote host when it is succeeded.
		stdout, stderr, _ := execAt(allBootServers[3], "sudo", "shutdown", "now")
		log.Info("boot-3 is stopped", map[string]interface{}{
			"stdout": string(stdout),
			"stderr": string(stderr),
		})

		By("Checking boot-3 machine state")
		Eventually(func() error {
			serial := fmt.Sprintf("%x", sha1.Sum([]byte("boot-3")))
			stdout := execSafeAt(bootServers[0], "sabactl", "machines", "get-state", serial)
			state := string(bytes.TrimSpace(stdout))
			if state != "unreachable" {
				return errors.New("boot-3 machine state is not unreachable: " + state)
			}
			return nil
		}).Should(Succeed())
	})
}
