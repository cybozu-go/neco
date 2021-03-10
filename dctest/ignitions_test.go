package dctest

import (
	"regexp"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	cryptDiskDir = "/dev/crypt-disk/by-path/"
)

func parentDev(str string) string {
	re := regexp.MustCompile(`^.*\((.*)\)$`)
	tmp := re.FindStringSubmatch(strings.TrimSpace(str))
	if len(tmp) != 2 {
		return ""
	}
	return tmp[1]
}

// testIgnitions tests for ignitions functions.
func testIgnitions() {
	targetSymlinks := []struct {
		symlink string
		diskDev string
	}{
		{
			symlink: cryptDiskDir + "pci-0000:00:09.0",
			diskDev: "vdc",
		},
		{
			symlink: cryptDiskDir + "pci-0000:00:0a.0",
			diskDev: "vdd",
		},
	}

	var ssNodeIP string
	It("should get SS Node IP address", func() {
		machines, err := getMachinesSpecifiedRole("ss")
		Expect(err).NotTo(HaveOccurred())
		ssNodeIP = machines[0].Spec.IPv4[0]
	})

	It("should create by-path based symlinks for encrypted devices", func() {
		By("checking the number of symlinks")
		stdout, stderr, err := execAt(bootServers[0], "ckecli", "ssh", "cybozu@"+ssNodeIP, "ls", cryptDiskDir)
		Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
		devices := strings.Fields(strings.TrimSpace(string(stdout)))
		Expect(devices).To(HaveLen(len(targetSymlinks)))

		for _, t := range targetSymlinks {
			By("checking the disk device of " + t.symlink)
			stdout, stderr, err = execAt(bootServers[0], "ckecli", "ssh", "cybozu@"+ssNodeIP, "--", "sudo", "dmsetup", "deps", t.symlink, "-o", "devname")
			Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
			cryptDev := parentDev(string(stdout))
			Expect(cryptDev).To(Equal(t.diskDev))
		}
	})
}
