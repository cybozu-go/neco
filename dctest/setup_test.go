package dctest

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/well"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// testSetup tests "neco setup"
func testSetup() {
	//  kubectl caches the results, but it takes time to write
	//  To speed up this caching, mount the directory to be cached by tmpfs.
	It("should create mount service", func() {
		mountUnit := `
[Unit]
Description=mount ,kube directory by tmpfs
Wants=network-online.target
After=network-online.target

[Mount]
What=tmpfs
Where=/home/cybozu/.kube
DirectoryMode=0777
Type=tmpfs
Options=strictatime,nosuid,nodev

[Install]
WantedBy=multi-user.target`
		mountServicePath := "/lib/systemd/system/home-cybozu-.kube.mount"

		for _, v := range bootServers {
			execSafeAt(v, "sudo", "systemd-analyze", "set-log-level", "debug")

			stdout, stderr, err := execAtWithInput(v, []byte(mountUnit), "sudo", "tee", mountServicePath)
			Expect(err).NotTo(HaveOccurred(), "host=%s, stdout=%s, stderr=%s, err=%v", v, stdout, stderr, err)
			execSafeAt(v, "test", "-f", mountServicePath)
			execSafeAt(v, "sudo", "systemctl", "enable", "home-cybozu-.kube.mount")
			execSafeAt(v, "sudo", "systemctl", "start", "home-cybozu-.kube.mount")
			execSafeAt(v, "systemctl", "-q", "is-active", "home-cybozu-.kube.mount")
		}
	})

	It("should complete on all boot servers", func() {
		env := well.NewEnvironment(context.Background())
		env.Go(func(ctx context.Context) error {
			stdout, stderr, err := execAtWithInput(
				bootServers[0], []byte(os.Getenv("GITHUB_TOKEN")), "sudo", "neco", "setup", "--no-revoke",
				"--proxy="+proxy, "0", "1", "2")
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
			stdout, stderr, err := execAtWithInput(
				bootServers[1], []byte(os.Getenv("GITHUB_TOKEN")), "sudo", "neco", "setup", "--no-revoke",
				"--proxy="+proxy, "0", "1", "2")
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
			stdout, stderr, err := execAtWithInput(
				bootServers[2], []byte(os.Getenv("GITHUB_TOKEN")), "sudo", "neco", "setup", "--no-revoke",
				"--proxy="+proxy, "0", "1", "2")
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
			execSafeAt(h, "test", "-f", "/lib/systemd/system/neco-rebooter.service")
			execSafeAt(h, "test", "-f", "/lib/systemd/system/sabakan-state-setter.service")
		}
	})

	It("should run services", func() {
		services := []string{
			"neco-updater.service",
			"neco-worker.service",
			"node-exporter.service",
			"etcd-backup.timer",
			neco.EtcdService + ".service",
			neco.VaultService + ".service",
		}
		Eventually(func() error {
			for _, h := range bootServers {
				for _, svc := range services {
					_, stderr, err := execAt(h, "systemctl", "-q", "is-active", svc)
					if err != nil {
						return fmt.Errorf("err: %v, stderr: %s", err, stderr)
					}
				}
			}
			return nil
		}, 1*time.Minute).Should(Succeed())
	})

	It("should complete updates", func() {
		By("Waiting for request to complete")
		waitRequestComplete("", true)

		By("Installing sshd_config and sudoers")
		for _, h := range bootServers {
			execSafeAt(h, "test", "-f", "/etc/ssh/sshd_config.d/neco.conf")
			execSafeAt(h, "sudo", "test", "-f", "/etc/sudoers.d/cybozu")
		}
	})
}
