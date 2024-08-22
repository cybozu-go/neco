package dctest

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cybozu-go/log"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/yaml"
)

const numActiveBootServers = 3

// runBeforeSuite is for Ginkgo BeforeSuite.
func runBeforeSuite() {
	fmt.Println("Preparing...")

	SetDefaultEventuallyPollingInterval(time.Second)
	SetDefaultEventuallyTimeout(10 * time.Minute)
	SetDefaultConsistentlyDuration(time.Second)
	SetDefaultConsistentlyPollingInterval(100 * time.Millisecond)

	data, err := os.ReadFile(machinesFile)
	Expect(err).NotTo(HaveOccurred())

	machines := struct {
		Racks []struct {
			Name    string `yaml:"name"`
			Workers struct {
				CS int `yaml:"cs"`
				SS int `yaml:"ss"`
			} `yaml:"workers"`
			Boot struct {
				Bastion string `yaml:"bastion"`
			} `yaml:"boot"`
		} `yaml:"racks"`
	}{}
	err = yaml.Unmarshal(data, &machines)
	Expect(err).NotTo(HaveOccurred(), "data=%s", data)

	for i, rack := range machines.Racks {
		addr := strings.Split(rack.Boot.Bastion, "/")[0]
		if i < numActiveBootServers {
			bootServers = append(bootServers, addr)
		}
		allBootServers = append(allBootServers, addr)
	}

	err = prepareSSHClients(allBootServers...)
	Expect(err).NotTo(HaveOccurred())

	// sync VM root filesystem to store newly generated SSH host keys.
	for h := range sshClients {
		execSafeAt(h, "sync")
	}

	log.DefaultLogger().SetOutput(GinkgoWriter)
}

// runBeforeSuiteInstall is for Ginkgo BeforeSuite, especially in bootstrap/functions test suites.
func runBeforeSuiteInstall() {
	// waiting for auto-config
	fmt.Println("waiting for auto-config has completed")
	Eventually(func() error {
		for _, host := range allBootServers {
			_, _, err := execAt(host, "test -f /tmp/auto-config-done")
			if err != nil {
				return fmt.Errorf("auto-config has not been completed. host: %s, err: %v", host, err)
			}
		}
		return nil
	}, 20*time.Minute).Should(Succeed())

	for _, host := range allBootServers {
		// Unhold cloud-init before purging netplan.io, otherwise pkgProblemResolver::Resolve generated breaks error happens
		execSafeAt(host, "sudo", "apt-mark", "unhold", "cloud-init")
		// on VMs, netplan.io should be purged after cloud-init completed
		execSafeAt(host, "sudo", "apt-get", "purge", "-y", "--autoremove", "netplan.io")
	}

	By("checking services on the boot servers are running")
	checkSystemdServicesOnBoot()

	// copy and install Neco deb package
	fmt.Println("installing Neco")
	f, err := os.Open(debFile)
	Expect(err).NotTo(HaveOccurred())
	defer f.Close()
	remoteFilename := filepath.Join("/tmp", filepath.Base(debFile))
	for _, host := range allBootServers {
		_, err := f.Seek(0, io.SeekStart)
		Expect(err).NotTo(HaveOccurred())
		stdout, stderr, err := execAtWithStream(host, f, "dd", "of="+remoteFilename)
		Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
		stdout, stderr, err = execAt(host, "sudo", "dpkg", "-i", remoteFilename)
		Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
	}

	fmt.Println("Begin tests...")
}

// runBeforeSuiteCopy is for Ginkgo BeforeSuite, especially in upgrade test suite.
func runBeforeSuiteCopy() {
	fmt.Println("distributing new neco package")
	f, err := os.Open(debFile)
	Expect(err).NotTo(HaveOccurred())
	defer f.Close()
	remoteFilename := filepath.Join("/tmp", filepath.Base(debFile))
	for _, host := range allBootServers {
		_, err := f.Seek(0, io.SeekStart)
		Expect(err).NotTo(HaveOccurred())
		stdout, stderr, err := execAtWithStream(host, f, "dd", "of="+remoteFilename)
		Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
	}

	fmt.Println("Begin tests...")
}
