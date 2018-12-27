package dctest

import (
	. "github.com/onsi/ginkgo"
)

// This must be the only top-level test container.
// Other tests and test containers must be listed in this.
var _ = Describe("Data center test", func() {
	Context("setup", testSetup)
	Context("initialize", testInit)
	Context("sabakan", testSabakan)
	// uploading contents to sabakan must be done after sabakan configuration.
	Context("contents", testContents)
	Context("upgrade", testUpgrade)
	Context("join/remove", testJoinRemove)
	Context("cke", testCKE)
	Context("coil", testCoil)
})
