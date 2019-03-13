package dctest

import (
	"fmt"
	"os"
	"path/filepath"
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
}

// RunBeforeSuiteInstall is for Ginkgo BeforeSuite, especially in bootstrap/functions test suites.
func RunBeforeSuiteInstall() {
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

	By("rebooting all boot servers")
	for _, host := range []string{boot0, boot1, boot2, boot3} {
		// Exit code is 255 when ssh is disconnected
		execAt(host, "sudo", "reboot")
	}

	By("waiting all boot servers are online")
	err := prepareSSHClients(boot0, boot1, boot2, boot3)
	Expect(err).NotTo(HaveOccurred())

	By("checking services on the boot servers are running after reboot")
	services := []string{
		"bird.service",
		"systemd-networkd.service",
		"chronyd.service",
		// chrony-wait.service can be started after reboot
		"chrony-wait.service",
	}
	Eventually(func() error {
		for _, host := range []string{boot0, boot1, boot2, boot3} {
			for _, service := range services {
				_, _, err := execAt(host, "systemctl", "-q", "is-active", service)
				if err != nil {
					return err
				}
			}
		}
		return nil
	}).Should(Succeed())

	// copy and install Neco deb package
	fmt.Println("installing Neco")
	f, err := os.Open(debFile)
	Expect(err).NotTo(HaveOccurred())
	defer f.Close()
	remoteFilename := filepath.Join("/tmp", filepath.Base(debFile))
	for _, host := range []string{boot0, boot1, boot2, boot3} {
		_, err := f.Seek(0, os.SEEK_SET)
		Expect(err).NotTo(HaveOccurred())
		_, _, err = execAtWithStream(host, f, "dd", "of="+remoteFilename)
		Expect(err).NotTo(HaveOccurred())
		stdout, stderr, err := execAt(host, "sudo", "dpkg", "-i", remoteFilename)
		Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
	}

	fmt.Println("Begin tests...")
}

// RunBeforeSuiteCopy is for Ginkgo BeforeSuite, especially in upgrade test suite.
func RunBeforeSuiteCopy() {
	fmt.Println("distributing new neco package")
	f, err := os.Open(debFile)
	Expect(err).NotTo(HaveOccurred())
	defer f.Close()
	remoteFilename := filepath.Join("/tmp", filepath.Base(debFile))
	for _, host := range []string{boot0, boot1, boot2, boot3} {
		_, err := f.Seek(0, os.SEEK_SET)
		Expect(err).NotTo(HaveOccurred())
		_, _, err = execAtWithStream(host, f, "dd", "of="+remoteFilename)
		Expect(err).NotTo(HaveOccurred())
	}

	fmt.Println("Begin tests...")
}
