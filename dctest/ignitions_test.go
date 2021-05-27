package dctest

import (
	"bufio"
	"regexp"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	systemDiskCount = 2

	cryptDiskDir = "/dev/crypt-disk/by-path/"
)

type diskDevInfo struct {
	pciSlot string
	diskDev string
}

/*
parses output from `ls -d /sys/block/`
filter device by vd* prefixed
example:
lrwxrwxrwx. 1 root root 0 May 24 09:22 vda -> ../devices/pci0000:00/0000:00:07.0/virtio4/block/vda
*/
func parseDiskDevInfo(str string) ([]diskDevInfo, error) {
	var info []diskDevInfo
	scanner := bufio.NewScanner(strings.NewReader(str))
	for scanner.Scan() {
		segs := strings.Split(strings.TrimSpace(scanner.Text()), "/")
		if len(segs) < 4 {
			continue
		}
		diskDev := segs[len(segs)-1]
		if !strings.HasPrefix(diskDev, "vd") {
			continue
		}
		info = append(info, diskDevInfo{
			pciSlot: segs[len(segs)-4],
			diskDev: diskDev,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return info, nil
}

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
	var ssNodeIP string
	It("should get SS Node IP address", func() {
		machines, err := getMachinesSpecifiedRole("ss")
		Expect(err).NotTo(HaveOccurred())
		ssNodeIP = machines[0].Spec.IPv4[0]
	})

	It("should create by-path based symlinks for encrypted devices", func() {
		By("checking the number of symlinks")
		stdout, stderr, err := execAt(bootServers[0], "ckecli", "ssh", "cybozu@"+ssNodeIP, "--", "ls", "-l", "/sys/block/")
		Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
		info, err := parseDiskDevInfo(string(stdout))
		Expect(err).NotTo(HaveOccurred(), "input=%s", string(stdout))

		stdout, stderr, err = execAt(bootServers[0], "ckecli", "ssh", "cybozu@"+ssNodeIP, "--", "ls", cryptDiskDir)
		Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
		devices := strings.Fields(strings.TrimSpace(string(stdout)))
		Expect(devices).To(HaveLen(len(info) - systemDiskCount))

		for _, i := range info[systemDiskCount:] {
			symlink := cryptDiskDir + "pci-" + i.pciSlot
			By("checking the disk device of " + symlink)
			stdout, stderr, err = execAt(bootServers[0], "ckecli", "ssh", "cybozu@"+ssNodeIP, "--", "sudo", "dmsetup", "deps", symlink, "-o", "devname")
			Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
			cryptDev := parentDev(string(stdout))
			Expect(cryptDev).To(Equal(i.diskDev))
		}
	})
}
