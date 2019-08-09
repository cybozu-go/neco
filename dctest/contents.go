package dctest

import (
	"bytes"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/sabakan"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/yaml"
)

// TestInitData executes "neco init-data"
func TestInitData() {
	It("should initialize data for sabakan and CKE", func() {
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

		By("initialize data for sabakan and CKE")
		execSafeAt(boot0, "neco", "cke", "weight", "set", "cs", "2")
		execSafeAt(boot0, "neco", "cke", "weight", "set", "ss", "1")
		stdout, stderr, err := execAt(boot0, "neco", "cke", "weight", "list")
		Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
		stdout, stderr, err = execAt(boot0, "neco", "cke", "weight", "get", "cs")
		Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
		Expect(string(bytes.TrimSpace(stdout))).To(Equal("2"))
		stdout, stderr, err = execAt(boot0, "neco", "cke", "weight", "get", "ss")
		Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
		Expect(string(bytes.TrimSpace(stdout))).To(Equal("1"))
		execSafeAt(boot0, "neco", "init-data")
		stdout, stderr, err = execAt(boot0, "ckecli", "sabakan", "get-template")
		Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
		ckeTemplate := cke.NewCluster()
		err = yaml.Unmarshal(stdout, ckeTemplate)
		Expect(err).NotTo(HaveOccurred())
		weight, err := strconv.ParseFloat(ckeTemplate.Nodes[1].Labels[sabakan.CKELabelWeight], 64)
		Expect(err).NotTo(HaveOccurred())
		Expect(weight).To(BeNumerically("==", 2.000000))
		// TODO: enable it after merge https://github.com/cybozu-go/neco/pull/430
		//weight, err = strconv.ParseFloat(ckeTemplate.Nodes[2].Labels[sabakan.CKELabelWeight], 64)
		//Expect(err).NotTo(HaveOccurred())
		//Expect(weight).To(BeNumerically("==", 1.000000))
	})
}
