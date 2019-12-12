package dctest

import (
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// TestIgnitions tests for ignitions functions.
func TestIgnitions() {
	It("should create by-path based name for encrypted devices", func() {
		machines, err := getMachinesSpecifiedRole("ss")
		Expect(err).NotTo(HaveOccurred())

		stdout, stderr, err := execAt(boot0, "ckecli", "ssh", "cybozu@"+machines[0].Spec.IPv4[0], "ls", "/dev/crypt-disk/by-path/")
		Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		devices := strings.Fields(strings.TrimSpace(string(stdout)))
		Expect(devices).ShouldNot(BeEmpty())
		for _, d := range devices {
			stdout, stderr, err = execAt(boot0, "ckecli", "ssh", "cybozu@"+machines[0].Spec.IPv4[0], "sudo", "cryptsetup", "status", filepath.Join("/dev/crypt-disk/by-path/", d))
			Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
		}
	})
}
