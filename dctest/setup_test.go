package dctest

import (
	"context"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/well"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// testSetup tests "neco setup"
func testSetup() {
	//  kubectl caches the results, but it takes time to write
	//  To speed up this caching, mount the directory to be cached by tmpfs.
	It("should create mount service", func() {
		mountUnit := `
[Unit]
Description=Mount .kube directory by tmpfs
Wants=network-online.target
After=network-online.target

[Service]
Type=oneshot
ExecStartPre=/bin/mkdir -p /home/cybozu/.kube
ExecStart=mount -t tmpfs -o size=5M tmpfs /home/cybozu/.kube`
		mountServicePath := "/lib/systemd/system/mount.service"

		for _, v := range bootServers {
			stdout, stderr, err := execAtWithInput(v, []byte(mountUnit), "sudo", "tee", mountServicePath)
			Expect(err).NotTo(HaveOccurred(), "host=%s, stdout=%s, stderr=%s, err=%v", v, stdout, stderr, err)
			execSafeAt(v, "test", "-f", mountServicePath)
			execSafeAt(v, "sudo", "systemctl", "enable", "mount")
			execSafeAt(v, "sudo", "systemctl", "start", "mount")
			execSafeAt(v, "systemctl", "-q", "is-active", "mount")
		}
	})

	It("should complete on all boot servers", func() {
		env := well.NewEnvironment(context.Background())
		env.Go(func(ctx context.Context) error {
			stdout, stderr, err := execAt(
				bootServers[0], "sudo", "neco", "setup", "--no-revoke", "--proxy="+proxy, "0", "1", "2")
			if err != nil {
				log.Error("neco setup failed", map[string]interface{}{
					"host":   "boot-0",
					"stdout": string(stdout),
					"stderr": string(stderr),
				})
				return err
			}
			return nil
		})
		env.Go(func(ctx context.Context) error {
			stdout, stderr, err := execAt(
				bootServers[1], "sudo", "neco", "setup", "--no-revoke", "--proxy="+proxy, "0", "1", "2")
			if err != nil {
				log.Error("neco setup failed", map[string]interface{}{
					"host":   "boot-1",
					"stdout": string(stdout),
					"stderr": string(stderr),
				})
				return err
			}
			return nil
		})
		env.Go(func(ctx context.Context) error {
			stdout, stderr, err := execAt(
				bootServers[2], "sudo", "neco", "setup", "--no-revoke", "--proxy="+proxy, "0", "1", "2")
			if err != nil {
				log.Error("neco setup failed", map[string]interface{}{
					"host":   "boot-2",
					"stdout": string(stdout),
					"stderr": string(stderr),
				})
				return err
			}
			return nil
		})
		env.Stop()

		Expect(env.Wait()).NotTo(HaveOccurred())
	})

	It("should install files", func() {
		for _, h := range bootServers {
			execSafeAt(h, "test", "-f", neco.NecoConfFile)
			execSafeAt(h, "test", "-f", neco.NecoCertFile)
			execSafeAt(h, "test", "-f", neco.NecoKeyFile)

			execSafeAt(h, "test", "-f", neco.EtcdBackupCertFile)
			execSafeAt(h, "test", "-f", neco.EtcdBackupKeyFile)
			execSafeAt(h, "test", "-f", neco.TimerFile("etcd-backup"))
			execSafeAt(h, "test", "-f", neco.ServiceFile("etcd-backup"))

			execSafeAt(h, "test", "-f", "/lib/systemd/system/neco-updater.service")
			execSafeAt(h, "test", "-f", "/lib/systemd/system/neco-worker.service")
			execSafeAt(h, "test", "-f", "/lib/systemd/system/node-exporter.service")
			execSafeAt(h, "test", "-f", "/lib/systemd/system/sabakan-state-setter.service")
		}
	})

	It("should run services", func() {
		for _, h := range bootServers {
			execSafeAt(h, "systemctl", "-q", "is-active", "neco-updater.service")
			execSafeAt(h, "systemctl", "-q", "is-active", "neco-worker.service")
			execSafeAt(h, "systemctl", "-q", "is-active", "node-exporter.service")
			execSafeAt(h, "systemctl", "-q", "is-active", "etcd-backup.timer")
			execSafeAt(h, "systemctl", "-q", "is-active", neco.EtcdService+".service")
			execSafeAt(h, "systemctl", "-q", "is-active", neco.VaultService+".service")
		}
	})

	It("should complete updates", func() {
		By("Waiting for request to complete")
		waitRequestComplete("")

		By("Installing sshd_config and sudoers")
		for _, h := range bootServers {
			execSafeAt(h, "test", "-f", "/etc/ssh/sshd_config.d/neco.conf")
			execSafeAt(h, "sudo", "test", "-f", "/etc/sudoers.d/cybozu")
		}
	})
}
