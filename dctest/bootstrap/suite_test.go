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
var _ = Describe("Test Neco bootstrap", dctest.BootstrapSuite)
