package dctest

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/cybozu-go/neco"
	sabakan "github.com/cybozu-go/sabakan/client"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// UploadContents executes "neco sabakan-upload"
func UploadContents() {
	It("should upload contents to sabakan", func() {
		data, err := ioutil.ReadFile("../secrets")
		switch {
		case err == nil:
			By("setting quay.io auth")
			// Don't print secret to console.
			// Don't print stdout/stderr of commands which handle secret.
			// The printed log in CircleCI is open to the public.
			passwd := string(bytes.TrimSpace(data))
			_, _, err = execAt(boot0, "env", "QUAY_USER=cybozu+neco_readonly", "neco", "config", "set", "quay-username")
			Expect(err).NotTo(HaveOccurred())
			_, _, err = execAt(boot0, "env", "QUAY_PASSWORD="+passwd, "neco", "config", "set", "quay-password")
			Expect(err).NotTo(HaveOccurred())
		case os.IsNotExist(err):
		default:
			Expect(err).NotTo(HaveOccurred())
		}

		By("uploading sabakan contents")
		execSafeAt(boot0, "neco", "sabakan-upload")
	})
}

// TestContents tests uploaded contents
func TestContents() {
	It("should check uploaded contents", func() {
		output := execSafeAt(boot0, "sabactl", "images", "index")
		index := new(sabakan.ImageIndex)
		err := json.Unmarshal(output, index)
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
		images := neco.SabakanPublicImages
		_, err = os.Stat("../secrets")
		secretExists := err == nil

		if secretExists {
			images = append(images, neco.SabakanPrivateImages...)
		}
		for _, name := range images {
			image, err := neco.CurrentArtifacts.FindContainerImage(name)
			Expect(err).NotTo(HaveOccurred())
			Expect(assets).To(ContainElement(neco.ImageAssetName(image)))
		}
		image, err := neco.CurrentArtifacts.FindContainerImage("sabakan")
		Expect(err).NotTo(HaveOccurred())
		Expect(assets).To(ContainElement(neco.CryptsetupAssetName(image)))
		for _, name := range []string{"hyperkube", "pause", "etcd", "coredns", "unbound"} {
			Expect(assets).To(ContainElement(MatchRegexp("^cybozu-%s-.*\\.img$", name)))
		}
	})
}
