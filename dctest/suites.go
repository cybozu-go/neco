package dctest

import . "github.com/onsi/ginkgo"

// BootstrapSuite is a test suite that tests initial setup of Neco
var BootstrapSuite = func() {
	Context("setup", TestSetup)
	Context("initialize", TestInit)
	Context("etcdpasswd", TestEtcdpasswd)
	Context("sabakan", TestSabakan)
	Context("machines", TestMachines)
	Context("init-data", TestInitData)
	Context("sabakan-state-setter", TestSabakanStateSetter)
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
	BootstrapSuite()
	Context("join/remove", TestJoinRemove)
	Context("parts failure", TestPartsFailure)
	Context("parts missing", TestPartsMissing)
	Context("reboot-all-nodes", TestRebootAllNodes)
}

// UpgradeSuite is a test suite that tests upgrading process works correctry
var UpgradeSuite = func() {
	Context("upgrade", TestUpgrade)
	Context("upgraded cke", TestCKE)
	Context("upgraded coil", TestCoil)
	Context("upgraded unbound", TestUnbound)
	Context("upgraded squid", TestSquid)
}
