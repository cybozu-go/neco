package dctest

import (
	"github.com/cybozu-go/neco"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// testBootIPSetter tests the behavior of boot-ip-setter in bootstrapping
func testBootIPSetter() {
	It("should set Virtual IPs to boot servers", func() {
		expectedDHCPServerHostname := map[string]string{
			"10.71.255.1": "gcp0-boot-0",
			"10.71.255.2": "gcp0-boot-1",
			"10.71.255.3": "gcp0-boot-2",
		}
		expectedActiveBootServerHostname := []string{
			"gcp0-boot-0",
			"gcp0-boot-1",
			"gcp0-boot-2",
		}
		checkBootServerVirtualIPs(expectedDHCPServerHostname, expectedActiveBootServerHostname)
	})
}

func checkBootServerVirtualIPs(expectedDHCPServerHostname map[string]string, expectedActiveBootServerHostname []string) {
	machines, err := getSabakanMachines("--without-role=boot")
	Expect(err).NotTo(HaveOccurred())

	By("checking dhcp server addresses")
	for _, m := range machines {
		nodeIP := m.Spec.IPv4[0]
		for _, vip := range neco.DHCPServerAddressList {
			if host := expectedDHCPServerHostname[vip]; host != "" {
				stdout, stderr, err := execAt(bootServers[0], "ckecli", "ssh", "cybozu@"+nodeIP, "--", "curl", "-m", "2", "-sS", "http://"+vip+":4192/hostname")
				Expect(err).NotTo(HaveOccurred(), "from=%s, to=%s, stdout=%s, stderr=%s", nodeIP, vip, stdout, stderr)
				Expect(string(stdout)).To(Equal(host))
			} else {
				stdout, stderr, err := execAt(bootServers[0], "ckecli", "ssh", "cybozu@"+nodeIP, "--", "curl", "-m", "2", "-sS", "http://"+vip+":4192/hostname")
				Expect(err).To(HaveOccurred(), "from=%s, to=%s, stdout=%s, stderr=%s", nodeIP, vip, stdout, stderr)
			}
		}
	}

	By("checking active boot server address")
	for _, m := range machines {
		nodeIP := m.Spec.IPv4[0]
		stdout, stderr, err := execAt(bootServers[0], "ckecli", "ssh", "cybozu@"+nodeIP, "--", "curl", "-m", "2", "-sS", "http://"+neco.VirtualIPAddrActiveBootServer+":4192/hostname")
		Expect(err).NotTo(HaveOccurred(), "from=%s, stdout=%s, stderr=%s", nodeIP, stdout, stderr)
		Expect(string(stdout)).To(BeElementOf(expectedActiveBootServerHostname))
	}
}
