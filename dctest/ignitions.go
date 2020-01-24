package dctest

import (
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// TestIgnitions tests for ignitions functions.
func TestIgnitions() {
	It("should create by-path based name for partitions on encrypted devices", func() {
		machines, err := getMachinesSpecifiedRole("ss")
		Expect(err).NotTo(HaveOccurred())

		cryptPartDir := "/dev/crypt-part/by-path/"

		By("checking count of symbols")
		stdout, stderr, err := execAt(boot0, "ckecli", "ssh", "cybozu@"+machines[0].Spec.IPv4[0], "ls", cryptPartDir)
		Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
		devices := strings.Fields(strings.TrimSpace(string(stdout)))
		Expect(devices).To(HaveLen(2))

		By("checking correspondence on device and partition")
		stdout, stderr, err = execAt(boot0, "ckecli", "ssh", "--", "cybozu@"+machines[0].Spec.IPv4[0], "sudo", "dmsetup", "deps", filepath.Join(cryptPartDir, "pci-0000\\:00\\:0a.0-p1"), "-o", "devname")
		Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
		Expect(string(stdout)).Should(ContainElement("crypt-vdc"))

		stdout, stderr, err = execAt(boot0, "ckecli", "ssh", "--", "cybozu@"+machines[0].Spec.IPv4[0], "sudo", "dmsetup", "deps", filepath.Join(cryptPartDir, "pci-0000\\:00\\:0b.0-p1"), "-o", "devname")
		Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
		Expect(string(stdout)).Should(ContainElement("crypt-vdd"))
	})
}
