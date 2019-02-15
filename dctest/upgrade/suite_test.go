package dctest

import (
	"os"
	"testing"

	"github.com/cybozu-go/neco/dctest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestDCtest(t *testing.T) {
	if os.Getenv("SSH_PRIVKEY") == "" {
		t.Skip("no SSH_PRIVKEY envvar")
	}

	RegisterFailHandler(Fail)
	RunSpecs(t, "Data Center test")
}

var _ = BeforeSuite(func() {
	dctest.RunBeforeSuite()
	dctest.RunBeforeSuiteCopy()
})

// This must be the only top-level test container.
// Other tests and test containers must be listed in this.
var _ = Describe("Test Neco upgrade", func() {
	Context("upgrade", dctest.TestUpgrade)
	Context("upgraded contents", dctest.TestContents)
	Context("upgraded cke", dctest.TestCKE)
	Context("upgraded coil", dctest.TestCoil)
	Context("upgraded unbound", dctest.TestUnbound)
	Context("upgraded squid", dctest.TestSquid)
})
