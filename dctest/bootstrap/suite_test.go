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
	dctest.RunBeforeSuiteInstall()
})

// This must be the only top-level test container.
// Other tests and test containers must be listed in this.
var _ = Describe("Test Neco bootstrap", func() {
	Context("setup", dctest.TestSetup)
	Context("initialize", dctest.TestInit)
	Context("etcdpasswd", dctest.TestEtcdpasswd)
	Context("sabakan", dctest.TestSabakan)
	Context("machines", dctest.TestMachines)
	Context("contents", dctest.UploadContents)
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
})
