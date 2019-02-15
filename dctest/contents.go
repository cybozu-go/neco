package dctest

import (
	"bytes"
	"io/ioutil"
	"os"

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
