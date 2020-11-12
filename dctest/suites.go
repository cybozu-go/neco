package dctest

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
)

// BootstrapSuite is a test suite that tests initial setup of Neco
var BootstrapSuite = func() {
	BeforeEach(func() {
		fmt.Printf("START: %s\n", time.Now().Format(time.RFC3339))
	})
	AfterEach(func() {
		fmt.Printf("END: %s\n", time.Now().Format(time.RFC3339))
	})

	Context("setup", TestSetup)
	Context("initialize", TestInit)
	Context("sabakan", TestSabakan)
	Context("machines", TestMachines)
	Context("init-data", TestInitData)
	Context("etcdpasswd", TestEtcdpasswd)
	Context("sabakan-state-setter", func() {
		TestSabakanStateSetter()
	})
	Context("ignitions", TestIgnitions)
	Context("cke", func() {
		TestCKESetup()
		TestCKE()
	})
	Context("coil", func() {
		TestCoilSetup()
		TestCoil()
	})
	Context("unbound", func() {
		TestUnbound()
	})
	Context("squid", func() {
		TestSquid()
	})
}

// FunctionsSuite is a test suite that tests a full set of functions of Neco in a single version
var FunctionsSuite = func() {
	BeforeEach(func() {
		fmt.Printf("START: %s\n", time.Now().Format(time.RFC3339))
	})
	AfterEach(func() {
		fmt.Printf("END: %s\n", time.Now().Format(time.RFC3339))
	})

	Context("join/remove", TestJoinRemove)
	Context("reboot-all-boot-servers", TestRebootAllBootServers)
	Context("reboot-all-nodes", TestRebootAllNodes)
}

// UpgradeSuite is a test suite that tests upgrading process works correctry
var UpgradeSuite = func() {
	BeforeEach(func() {
		fmt.Printf("START: %s\n", time.Now().Format(time.RFC3339))
	})
	AfterEach(func() {
		fmt.Printf("END: %s\n", time.Now().Format(time.RFC3339))
	})

	Context("sabakan-state-setter", func() {
		TestSabakanStateSetter()
	})
	Context("upgrade", TestUpgrade)
	// TODO: Remove TestRebootAllNodes from the upgrade suite after the following PR is released.
	// https://github.com/cybozu-go/neco/pull/1180
	Context("reboot-all-nodes", TestRebootAllNodes)
	Context("upgraded cke", func() {
		TestCKE()
	})
	Context("upgraded coil", TestCoil)
	Context("upgraded unbound", TestUnbound)
	Context("upgraded squid", TestSquid)
}
