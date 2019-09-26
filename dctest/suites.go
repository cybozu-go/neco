package dctest

import . "github.com/onsi/ginkgo"

// BootstrapSuite is a test suite that tests initial setup of Neco
var BootstrapSuite = func() {
	// cs x 6 + ss x 1 = 7
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
	Context("serf-tags-proxy", TestSerfTagsProxy)
}

// UpgradeSuite is a test suite that tests upgrading process works correctry
var UpgradeSuite = func() {
	// cs x 6 + ss x 1 = 7
	availableNodes := 7
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
