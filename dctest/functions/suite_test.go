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
var _ = Describe("Test Neco functions", func() {
	Context("setup", dctest.TestSetup)
	Context("initialize", dctest.TestInit)
	Context("etcdpasswd", dctest.TestEtcdpasswd)
	Context("sabakan", dctest.TestSabakan)
	// uploading contents to sabakan must be done after sabakan configuration.
	Context("contents", func() {
		dctest.UploadContents()
		dctest.TestContents()
	})
	Context("upgrade", dctest.TestUpgrade)
	Context("join/remove", dctest.TestJoinRemove)
	Context("cke", func() {
		dctest.TestCKE("0.0.2")
	})
	Context("coil", dctest.TestCoil)
	Context("unbound", dctest.TestUnbound)
	Context("squid", dctest.TestSquid)
})
