package dctest

import (
	"fmt"
	"time"

	"github.com/cybozu-go/log"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// RunBeforeSuite is for Ginkgo BeforeSuite.
func RunBeforeSuite() {
	fmt.Println("Preparing...")

	SetDefaultEventuallyPollingInterval(time.Second)
	SetDefaultEventuallyTimeout(10 * time.Minute)

	err := prepareSSHClients(boot0, boot1, boot2, boot3)
	Expect(err).NotTo(HaveOccurred())

	// sync VM root filesystem to store newly generated SSH host keys.
	for h := range sshClients {
		execSafeAt(h, "sync")
	}

	log.DefaultLogger().SetOutput(GinkgoWriter)

	// waiting for auto-config
	fmt.Println("waiting for auto-config has completed")
	Eventually(func() error {
		for _, host := range []string{boot0, boot1, boot2, boot3} {
			_, _, err := execAt(host, "test -f /tmp/auto-config-done")
			if err != nil {
				return err
			}
		}
		return nil
	}).Should(Succeed())

	fmt.Println("Begin tests...")
}
