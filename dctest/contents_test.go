package dctest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/cybozu-go/neco"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

const ckeLabelWeight = "cke.cybozu.com/weight"

// testInitData executes "neco init-data"
func testInitData() {
	if os.Getenv("NOT_NECO_CI") != "" && os.Getenv("NECO_CI_BLOB_CACHE_URL") != "" {
		It("should upload initial kernel images from blob cache to sabakan", func() {
			osImage := &neco.CurrentArtifacts.OSImage
			kernelUrl, initrdUrl := osImage.URLs()
			cmd := exec.Command("./upload-initial-kernel-image.sh", kernelUrl, initrdUrl, osImage.Version)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err := cmd.Run()
			Expect(err).NotTo(HaveOccurred())
		})
	}

	It("should initialize data for sabakan and CKE", func() {
		By("setting external IP address block")
		execSafeAt(bootServers[0], "neco", "config", "set", "external-ip-address-block", externalIPBlock)

		By("setting LB address block for default")
		execSafeAt(bootServers[0], "neco", "config", "set", "lb-address-block-default", lbAddressBlockDefault)

		By("setting LB address block for bastion")
		execSafeAt(bootServers[0], "neco", "config", "set", "lb-address-block-bastion", lbAddressBlockBastion)

		By("setting LB address block for internet")
		execSafeAt(bootServers[0], "neco", "config", "set", "lb-address-block-internet", lbAddressBlockInternet)

		// By("setting LB address block for internet-cn")
		// execSafeAt(bootServers[0], "neco", "config", "set", "lb-address-block-internet-cn", lbAddressBlockInternetCN)

		By("initialize data for sabakan and CKE")
		cs, err := getMachinesSpecifiedRole("cs")
		Expect(err).NotTo(HaveOccurred())
		ss, err := getMachinesSpecifiedRole("ss")
		Expect(err).NotTo(HaveOccurred())
		// substruct the number of control planes
		csweight := len(cs) - 3
		ssweight := len(ss)
		Expect(csweight).Should(BeNumerically(">", 0))
		Expect(ssweight).Should(BeNumerically(">", 0))
		execSafeAt(bootServers[0], "neco", "cke", "weight", "set", "cs", strconv.Itoa(csweight))
		execSafeAt(bootServers[0], "neco", "cke", "weight", "set", "ss", strconv.Itoa(ssweight))
		stdout, stderr, err := execAt(bootServers[0], "neco", "cke", "weight", "list")
		Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
		var result map[string]int
		err = json.Unmarshal(stdout, &result)
		Expect(err).NotTo(HaveOccurred(), "data=%s", stdout)
		v, ok := result["cs"]
		Expect(ok).To(BeTrue())
		Expect(v).To(Equal(csweight))
		v, ok = result["ss"]
		Expect(ok).To(BeTrue())
		Expect(v).To(Equal(ssweight))
		stdout, stderr, err = execAt(bootServers[0], "neco", "cke", "weight", "get", "cs")
		Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
		Expect(string(bytes.TrimSpace(stdout))).To(Equal(strconv.Itoa(csweight)))
		stdout, stderr, err = execAt(bootServers[0], "neco", "cke", "weight", "get", "ss")
		Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
		Expect(string(bytes.TrimSpace(stdout))).To(Equal(strconv.Itoa(ssweight)))
		for _, bootServer := range allBootServers {
			execSafeAt(bootServer, "sudo", "sed", "-i", "'s/#GCPONLY //g'", "/usr/share/neco/cke-template.yml")
		}
		execSafeAt(bootServers[0], "neco", "cke", "update")
		stdout, stderr, err = execAt(bootServers[0], "ckecli", "sabakan", "get-template")
		Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		var ckeTemplate map[string]interface{}
		err = yaml.Unmarshal(stdout, &ckeTemplate)
		Expect(err).NotTo(HaveOccurred(), "data=%s", stdout)
		nodes, found, err := unstructured.NestedSlice(ckeTemplate, "nodes")
		Expect(found).To(BeTrue())
		Expect(err).NotTo(HaveOccurred())

		labels, found, err := unstructured.NestedStringMap(nodes[1].(map[string]interface{}), "labels")
		Expect(found).To(BeTrue())
		Expect(err).NotTo(HaveOccurred())
		weight, err := strconv.ParseFloat(labels[ckeLabelWeight], 64)
		Expect(err).NotTo(HaveOccurred(), "data=%s", labels[ckeLabelWeight])
		Expect(weight).To(BeNumerically("==", csweight))

		labels, found, err = unstructured.NestedStringMap(nodes[2].(map[string]interface{}), "labels")
		Expect(found).To(BeTrue())
		Expect(err).NotTo(HaveOccurred())
		weight, err = strconv.ParseFloat(labels[ckeLabelWeight], 64)
		Expect(err).NotTo(HaveOccurred(), "data=%s", labels[ckeLabelWeight])
		Expect(weight).To(BeNumerically("==", ssweight))

		Eventually(func() error {
			stdout, stderr, err := execAt(bootServers[0], "neco", "init-data")
			if err != nil {
				return fmt.Errorf("neco init-data failed; err: %s, stdout: %s, stderr: %s", err, stdout, stderr)
			}
			return nil
		}).Should(Succeed())
	})
}
