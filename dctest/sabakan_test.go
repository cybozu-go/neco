package dctest

import (
	"encoding/json"
	"io/ioutil"

	"github.com/cybozu-go/neco"
	sabakan "github.com/cybozu-go/sabakan/client"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func testSabakan() {
	It("should success initialize sabakan data", func() {
		execSafeAt(boot0, "sabactl", "ipam", "set", "-f", "/mnt/ipam.json")
		execSafeAt(boot0, "sabactl", "dhcp", "set", "-f", "/mnt/dhcp.json")
		execSafeAt(boot0, "sabactl", "machines", "create", "-f", "/mnt/machines.json")
		execSafeAt(boot0, "sabactl", "kernel-params", "set", "console=ttyS0")

		secrets, err := ioutil.ReadFile("secrets")
		Expect(err).NotTo(HaveOccurred())
		err = execAtWithInput(boot0, secrets, "dd", "of=secrets.sh")
		Expect(err).NotTo(HaveOccurred())
		script, err := ioutil.ReadFile("with_secrets.sh")
		Expect(err).NotTo(HaveOccurred())
		err = execAtWithInput(boot0, script, "dd", "of=secrets.sh", "oflag=append", "conv=notrunc")
		Expect(err).NotTo(HaveOccurred())
		execSafeAt(boot0, "sh", "-e", "secrets.sh")

		output := execSafeAt(boot0, "sabactl", "images", "index")
		index := new(sabakan.ImageIndex)
		err = json.Unmarshal(output, index)
		Expect(err).NotTo(HaveOccurred())
		Expect(index.Find(neco.CurrentArtifacts.CoreOS.Version)).NotTo(BeNil())

		output = execSafeAt(boot0, "dpkg-query", "--showformat=\\${Version}", "-W", neco.NecoPackageName)
		necoVersion := string(output)
		output = execSafeAt(boot0, "sabactl", "ignitions", "get", "worker")
		var ignInfo []*sabakan.IgnitionInfo
		err = json.Unmarshal(output, &ignInfo)
		Expect(err).NotTo(HaveOccurred())
		Expect(ignInfo).To(HaveLen(1))
		Expect(ignInfo[0].ID).To(Equal(necoVersion))

		output = execSafeAt(boot0, "sabactl", "assets", "index")
		var assets []string
		err = json.Unmarshal(output, &assets)
		Expect(err).NotTo(HaveOccurred())
		for _, image := range neco.SystemContainers {
			Expect(assets).To(ContainElement(neco.ACIAssetName(image)))
		}
		for _, name := range []string{"serf", "omsa", "coil", "squid"} {
			image, err := neco.CurrentArtifacts.FindContainerImage(name)
			Expect(err).NotTo(HaveOccurred())
			Expect(assets).To(ContainElement(neco.ImageAssetName(image)))
		}
		image, err := neco.CurrentArtifacts.FindContainerImage("sabakan")
		Expect(err).NotTo(HaveOccurred())
		Expect(assets).To(ContainElement(neco.CryptsetupAssetName(image)))
	})

	It("should update machine state in sabakan", func() {
		// Restart serf after machine registered to update state in sabakan
		for _, host := range []string{boot0, boot1, boot2} {
			execSafeAt(host, "sudo", "systemctl", "restart", "serf.service")
		}

		for _, ip := range []string{"10.69.0.3", "10.69.0.195", "10.69.1.131"} {
			Eventually(func() []byte {
				return execSafeAt(boot0, "sabactl", "machines", "get", "-ipv4", ip)
			}).Should(ContainSubstring(`"state": "healthy"`))
		}
	})

	It("should upload k8s-related containers", func() {
		output := execSafeAt(boot0, "sabactl", "assets", "index")
		var assets []string
		err := json.Unmarshal(output, &assets)
		Expect(err).NotTo(HaveOccurred())
		for _, name := range []string{"hyperkube", "pause", "etcd", "coredns", "unbound"} {
			Expect(assets).To(ContainElement(MatchRegexp("^cybozu-%s-.*\\.img$", name)))
		}
	})
}
