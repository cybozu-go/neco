package dctest

import . "github.com/onsi/ginkgo"

// BootstrapSuite is a test suite that tests initial setup of Neco
var BootstrapSuite = func() {
	availableNodes := 7
	Context("setup", TestSetup)
	Context("initialize", TestInit)
	Context("sabakan", TestSabakan)
	Context("machines", TestMachines)
	Context("init-data", TestInitData)
	Context("etcdpasswd", TestEtcdpasswd)
	Context("sabakan-state-setter", func() {
		TestSabakanStateSetter(availableNodes)
	})
	Context("cke", func() {
		TestCKESetup()
		TestCKE(availableNodes)
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
	Context("join/remove", TestJoinRemove)
	Context("reboot-all-boot-servers", TestRebootAllBootServers)
	Context("reboot-all-nodes", TestRebootAllNodes)
}

// UpgradeSuite is a test suite that tests upgrading process works correctry
var UpgradeSuite = func() {
	// TODO: Please update it to 7 after merge branch ignition-for-SS
	availableNodes := 6
	Context("sabakan-state-setter", func() {
		TestSabakanStateSetter(availableNodes)
	})
	Context("upgrade", TestUpgrade)
	Context("upgraded cke", func() {
		TestCKE(availableNodes)
	})
	Context("upgraded coil", TestCoil)
	Context("upgraded unbound", TestUnbound)
	Context("upgraded squid", TestSquid)
}
