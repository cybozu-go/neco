package dctest

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// testRebootAllBootServers tests all boot servers are normal after reboot
func testRebootAllBootServers() {
	It("runs systemd service on all boot servers after reboot", func() {
		By("rebooting all boot servers")
		for _, host := range bootServers {
			// Exit code is 255 when ssh is disconnected
			execAt(host, "sudo", "reboot")
		}

		By("waiting all boot servers are online")
		err := prepareSSHClients(bootServers...)
		Expect(err).NotTo(HaveOccurred())

		By("checking services on the boot servers are running after reboot")
		checkSystemdServicesOnBoot()

		By("checking sabakan is available")
		Eventually(func() error {
			stdout, stderr, err := execAt(bootServers[0], "sabactl", "machines", "get")
			if err != nil {
				return fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
			}
			return nil
		}).Should(Succeed())

		// .kube directory is mounted by tmpfs, so reissuing config file is necessary
		By("generating kubeconfig for cluster admin")
		Eventually(func() error {
			_, stderr, err := execAt(bootServers[0], "ckecli", "kubernetes", "issue", ">", ".kube/config")
			if err != nil {
				return fmt.Errorf("err: %v, stderr: %s", err, stderr)
			}
			return nil
		}).Should(Succeed())
	})
}

// testRebootGracefully tests graceful reboot of workers
func testRebootGracefully() {
	It("can reboot all workers gracefully", func() {
		workersBefore, err := getSerfWorkerMembers()
		Expect(err).NotTo(HaveOccurred())

		execSafeAt(bootServers[0], "sh", "-c", "yes | neco reboot-worker")
		Eventually(func() error {
			stdout, stderr, err := execAt(bootServers[0], "ckecli", "rq", "list")
			if err != nil {
				return fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
			}
			if string(stdout) != "null\n" {
				return fmt.Errorf("reboot-queue is not processed")
			}
			return nil
		}, 30*time.Minute).Should(Succeed())

		workersAfter, err := getSerfWorkerMembers()
		Expect(err).NotTo(HaveOccurred())
		for _, before := range workersBefore.Members {
			for _, after := range workersAfter.Members {
				if before.Name == after.Name {
					Expect(after.Tags["uptime"]).NotTo(Equal(before.Tags["uptime"]))
				}
			}
		}
	})
}
