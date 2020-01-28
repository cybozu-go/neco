package dctest

import (
	"regexp"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func parentDev(str string) string {
	re := regexp.MustCompile(`^.*\((.*)\)$`)
	tmp := re.FindStringSubmatch(strings.TrimSpace(str))
	if len(tmp) != 2 {
		return ""
	}
	return tmp[1]
}

// TestIgnitions tests for ignitions functions.
func TestIgnitions() {
	const cryptPartDir = "/dev/crypt-part/by-path/"

	targetSymlinks := []struct {
		symlink string
		diskDev string
	}{
		{
			symlink: cryptPartDir + "pci-0000:00:0a.0-p1",
			diskDev: "vdc",
		},
		{
			symlink: cryptPartDir + "pci-0000:00:0b.0-p1",
			diskDev: "vdd",
		},
	}

	It("should create by-path based symlinks for partitions on encrypted devices", func() {
		By("getting SS Node IP address")
		machines, err := getMachinesSpecifiedRole("ss")
		Expect(err).NotTo(HaveOccurred())
		ssNodeIP := machines[0].Spec.IPv4[0]

		By("checking the number of symlinks")
		stdout, stderr, err := execAt(boot0, "ckecli", "ssh", "cybozu@"+ssNodeIP, "ls", cryptPartDir)
		Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
		devices := strings.Fields(strings.TrimSpace(string(stdout)))
		Expect(devices).To(HaveLen(len(targetSymlinks)))

		for _, t := range targetSymlinks {
			By("checking the disk device of " + t.symlink)

			By("getting the crypt volume")
			stdout, stderr, err = execAt(boot0, "ckecli", "ssh", "cybozu@"+ssNodeIP, "--", "sudo", "dmsetup", "deps", t.symlink, "-o", "devname")
			Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
			cryptDev := parentDev(string(stdout))
			Expect(cryptDev).NotTo(BeEmpty())

			By("getting the disk device")
			stdout, stderr, err = execAt(boot0, "ckecli", "ssh", "cybozu@"+ssNodeIP, "--", "sudo", "dmsetup", "deps", cryptDev, "-o", "devname")
			Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
			actualDiskDev := parentDev(string(stdout))
			Expect(actualDiskDev).To(Equal(t.diskDev))
		}
	})
}
