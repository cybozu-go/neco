package dctest

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

const ckeLabelWeight = "cke.cybozu.com/weight"

// testInitData executes "neco init-data"
func testInitData() {
	It("should initialize data for sabakan and CKE", func() {
		data, err := ioutil.ReadFile("../secrets")
		switch {
		case err == nil:
			By("setting quay.io auth")
			// Don't print secret to console.
			// Don't print stdout/stderr of commands which handle secret.
			// The printed log in CircleCI is open to the public.
			passwd := string(bytes.TrimSpace(data))
			_, _, err = execAt(bootServers[0], "env", "QUAY_USER=cybozu+neco_readonly", "neco", "config", "set", "quay-username")
			Expect(err).NotTo(HaveOccurred())
			_, _, err = execAt(bootServers[0], "env", "QUAY_PASSWORD="+passwd, "neco", "config", "set", "quay-password")
			Expect(err).NotTo(HaveOccurred())
		case os.IsNotExist(err):
		default:
			Expect(err).NotTo(HaveOccurred())
		}

		By("setting external IP address block")
		execSafeAt(bootServers[0], "neco", "config", "set", "external-ip-address-block", externalIPBlock)

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
		Expect(err).NotTo(HaveOccurred())
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
		Expect(err).NotTo(HaveOccurred())
		nodes, found, err := unstructured.NestedSlice(ckeTemplate, "nodes")
		Expect(found).To(BeTrue())
		Expect(err).NotTo(HaveOccurred())

		labels, found, err := unstructured.NestedStringMap(nodes[1].(map[string]interface{}), "labels")
		Expect(found).To(BeTrue())
		Expect(err).NotTo(HaveOccurred())
		weight, err := strconv.ParseFloat(labels[ckeLabelWeight], 64)
		Expect(err).NotTo(HaveOccurred())
		Expect(weight).To(BeNumerically("==", csweight))

		labels, found, err = unstructured.NestedStringMap(nodes[2].(map[string]interface{}), "labels")
		Expect(found).To(BeTrue())
		Expect(err).NotTo(HaveOccurred())
		weight, err = strconv.ParseFloat(labels[ckeLabelWeight], 64)
		Expect(err).NotTo(HaveOccurred())
		Expect(weight).To(BeNumerically("==", ssweight))

		execSafeAt(bootServers[0], "neco", "init-data")
	})
}
