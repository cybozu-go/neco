package dctest

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
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
	runBeforeSuite()

	switch testSuite {
	case "bootstrap":
		runBeforeSuiteInstall()
	case "upgrade":
		runBeforeSuiteCopy()
	}
})

// This must be the only top-level test container.
// Other tests and test containers must be listed in this.
var _ = Describe("Neco", func() {
	switch testSuite {
	case "bootstrap":
		bootstrapSuite()
	case "functions":
		functionsSuite()
	case "upgrade":
		upgradeSuite()
	case "reboot-worker":
		rebootWorkerSuite()
	}
})

// bootstrapSuite is a test suite that tests initial setup of Neco
var bootstrapSuite = func() {
	Context("setup", testSetup)
	Context("initialize", testInit)
	Context("sabakan", testSabakan)
	Context("machines", testMachines)
	Context("init-data", testInitData)
	Context("etcdpasswd", testEtcdpasswd)
	Context("sabakan-state-setter", testSabakanStateSetter)
	Context("boot-ip-setter", testBootIPSetter)
	Context("ignitions", testIgnitions)
	Context("cke", func() {
		testCKESetup()
		testCKE()
		testCKEBackupMetrics()
	})
	Context("neco-rebooter", testNecoRebooter)
	Context("coil", func() {
		testCoilSetup()
		testCoil()
	})
	Context("cilium", testCilium)
	Context("unbound", testUnbound)
	Context("squid", testSquid)
	Context("node DNS", testNodeDNS)
	Context("l4lb", testL4LB)
	Context("tools", testTools)
}

// functionsSuite is a test suite that tests a full set of functions of Neco in a single version
var functionsSuite = func() {
	Context("join/remove", testJoinRemove)
	Context("reboot-all-boot-servers", testRebootAllBootServers)
	Context("retire-server", testRetireServer)
}

// rebootWorkerSuite is a test suite that tests neco reboot-worker command
var rebootWorkerSuite = func() {
	Context("cke-reboot-gracefully", testCKERebootGracefully)
	Context("neco-rebooter-reboot-gracefully", testNecoRebooterRebootGracefully)
}

// upgradeSuite is a test suite that tests upgrading process works correctry
var upgradeSuite = func() {
	Context("upgrade", testUpgrade)
	Context("upgrade sabakan-state-setter", testSabakanStateSetter)
	Context("neco-rebooter", testNecoRebooter)
	Context("upgraded cke", testCKE)
	Context("upgraded coil", testCoil)
	Context("upgraded cilium", testCilium)
	Context("upgraded unbound", testUnbound)
	Context("upgraded squid", testSquid)
	Context("upgraded l4lb", testL4LB)
}
