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

var _ = BeforeSuite(dctest.RunBeforeSuite)

// This must be the only top-level test container.
// Other tests and test containers must be listed in this.
var _ = Describe("Test Neco upgrade", func() {
	Context("setup", dctest.TestSetup)
	Context("initialize", dctest.TestInit)
	Context("etcdpasswd", dctest.TestEtcdpasswd)
	Context("sabakan", dctest.TestSabakan)
	// uploading contents to sabakan must be done after sabakan configuration.
	// dctest.TestContents cannot be here because the contents are old due to the old Neco deb.
	Context("contents", dctest.UploadContents)
	Context("machines", dctest.TestMachines)
	Context("cke", func() {
		dctest.TestCKESetup()
		dctest.TestCKE()
	})
	Context("coil", func() {
		dctest.TestCoilSetup()
		dctest.TestCoil()
	})
	Context("unbound", func() {
		dctest.TestUnboundSetup()
		dctest.TestUnbound()
	})
	Context("squid", func() {
		dctest.TestSquidSetup()
		dctest.TestSquid()
	})
	Context("upgrade", dctest.TestUpgrade)
	Context("upgraded contents", dctest.TestContents)
	Context("upgraded cke", dctest.TestCKE)
	Context("upgraded coil", dctest.TestCoil)
	Context("upgraded unbound", dctest.TestUnbound)
	Context("upgraded squid", dctest.TestSquid)
})
