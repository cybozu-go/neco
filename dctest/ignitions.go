package dctest

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	cryptPartDir = "/dev/crypt-part/by-path/"
	partUUDIFile = "../output/partition-uuid.txt"
)

func partUUIDFileExists() bool {
	if _, err := os.Stat(partUUDIFile); os.IsNotExist(err) {
		return false
	}
	return true
}

func savePartUUID(partUUID map[string]string) {
	var buf string
	for symlink, uuid := range partUUID {
		buf += fmt.Sprintf("%s,%s\n", symlink, uuid)
	}
	err := ioutil.WriteFile(partUUDIFile, []byte(buf), os.FileMode(0644))
	Expect(err).NotTo(HaveOccurred())
}

func loadPartUUID() map[string]string {
	buf, err := ioutil.ReadFile(partUUDIFile)
	Expect(err).NotTo(HaveOccurred())

	partUUID := make(map[string]string)
	scanner := bufio.NewScanner(bytes.NewReader(buf))
	for scanner.Scan() {
		slice := strings.Split(scanner.Text(), ",")
		partUUID[slice[0]] = slice[1]
	}
	return partUUID
}

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

	var ssNodeIP string
	It("should get SS Node IP address", func() {
		machines, err := getMachinesSpecifiedRole("ss")
		Expect(err).NotTo(HaveOccurred())
		ssNodeIP = machines[0].Spec.IPv4[0]
	})

	It("should create by-path based symlinks for partitions on encrypted devices", func() {
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

	It("should not overwrite the partitions after reboot", func() {
		if !partUUIDFileExists() {
			Skip("this step does not execute at the first bootstrap")
		}

		By("loading the partition UUIDs")
		storedPartUUID := loadPartUUID()
		for symlink, uuid := range storedPartUUID {
			fmt.Printf("%s: %s\n", symlink, uuid)
		}

		By("checking the partition UUIDs")
		for _, t := range targetSymlinks {
			stored := storedPartUUID[t.symlink]
			stdout, stderr, err := execAt(boot0, "ckecli", "ssh", "cybozu@"+ssNodeIP, "--", "sudo", "blkid", "-s", "PARTUUID", "-o", "value", t.symlink)
			Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
			uuid := strings.TrimSpace(string(stdout))
			Expect(uuid).To(Equal(stored))
		}
	})

	It("should save the partition UUIDs", func() {
		if partUUIDFileExists() {
			Skip("this step only executes at the first bootstrap")
		}

		By("getting the partition UUIDs")
		partUUID := make(map[string]string)
		for _, t := range targetSymlinks {
			stdout, stderr, err := execAt(boot0, "ckecli", "ssh", "cybozu@"+ssNodeIP, "--", "sudo", "blkid", "-s", "PARTUUID", "-o", "value", t.symlink)
			Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
			uuid := strings.TrimSpace(string(stdout))
			Expect(uuid).NotTo(BeEmpty())
			partUUID[t.symlink] = uuid
		}

		By("saving the partition UUIDs")
		for symlink, uuid := range partUUID {
			fmt.Printf("%s: %s\n", symlink, uuid)
		}
		savePartUUID(partUUID)
	})
}
