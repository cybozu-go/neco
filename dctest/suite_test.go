package dctest

import (
	"fmt"
	"testing"
	"time"

	"github.com/cybozu-go/log"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestDCtest(t *testing.T) {
	if len(sshKeyFile) == 0 {
		t.Skip("no SSH_PRIVKEY envvar")
	}
	RegisterFailHandler(Fail)
	RunSpecs(t, "Data Center test")
}

var _ = BeforeSuite(func() {
	fmt.Println("Preparing...")

	SetDefaultEventuallyPollingInterval(time.Second)
	SetDefaultEventuallyTimeout(6 * time.Minute)

	err := prepareSSHClients(boot0, boot1, boot2)
	Expect(err).NotTo(HaveOccurred())

	// sync VM root filesystem to store newly generated SSH host keys.
	for h := range sshClients {
		execSafeAt(h, "sync")
	}

	log.DefaultLogger().SetOutput(GinkgoWriter)

	// waiting for auto-config
	fmt.Println("waiting for auto-config has completed")
	Eventually(func() error {
		for _, host := range []string{boot0, boot1, boot2} {
			_, _, err := execAt(host, "test -f /tmp/auto-config-done")
			if err != nil {
				return err
			}
		}
		return nil
	}).Should(Succeed())

	fmt.Println("Begin tests...")
})
