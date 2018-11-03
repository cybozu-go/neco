package dctest

import (
	"context"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/well"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// test container for "neco setup"
func testSetup() {
	It("should complete on all boot servers", func(done Done) {
		env := well.NewEnvironment(context.Background())
		env.Go(func(ctx context.Context) error {
			stdout, stderr, err := execAt(
				boot0, "sudo", "/mnt/neco", "setup", "--no-revoke", "0", "1", "2")
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
				boot1, "sudo", "/mnt/neco", "setup", "--no-revoke", "0", "1", "2")
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
				boot2, "sudo", "/mnt/neco", "setup", "--no-revoke", "0", "1", "2")
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
		close(done)
	}, 300)

	It("should install files", func() {
		for _, h := range []string{boot0, boot1, boot2} {
			execSafeAt(h, "test", "-x", neco.NecoBin)
			execSafeAt(h, "test", "-f", neco.NecoConfFile)
			execSafeAt(h, "test", "-f", neco.NecoCertFile)
			execSafeAt(h, "test", "-f", neco.NecoKeyFile)

			execSafeAt(h, "test", "-f", neco.EtcdBackupCertFile)
			execSafeAt(h, "test", "-f", neco.EtcdBackupKeyFile)
			execSafeAt(h, "test", "-f", neco.TimerFile("etcd-backup"))
			execSafeAt(h, "test", "-f", neco.ServiceFile("etcd-backup"))
		}
	})
}
