package dctest

import . "github.com/onsi/ginkgo"

// BootstrapSuite is a test suite that tests initial setup of Neco
var BootstrapSuite = func() {
	Context("setup", TestSetup)
	Context("initialize", TestInit)
	Context("etcdpasswd", TestEtcdpasswd)
	Context("sabakan", TestSabakan)
	Context("machines", TestMachines)
	Context("contents", UploadContents)
	Context("cke", func() {
		TestCKESetup()
		TestCKE()
	})
	Context("coil", func() {
		TestCoilSetup()
		TestCoil()
	})
	Context("unbound", func() {
		TestUnboundSetup()
		TestUnbound()
	})
	Context("squid", func() {
		TestSquidSetup()
		TestSquid()
	})
}

// FunctionsSuite is a test suite that tests a full set of functions of Neco in a single version
var FunctionsSuite = func() {
	BootstrapSuite()
	Context("join/remove", TestJoinRemove)
}

// RebootSuite is a test suite that tests disaster recovery scenario
var RebootSuite = func() {
	FunctionsSuite()
	Context("reboot-all-nodes", TestRebootAllNodes)
}
