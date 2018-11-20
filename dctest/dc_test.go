package dctest

import (
	. "github.com/onsi/ginkgo"
)

// This must be the only top-level test container.
// Other tests and test containers must be listed in this.
var _ = Describe("Data center test", func() {
	Context("setup", testSetup)
	Context("worker", testWorker)
	Context("join/remove", testJoinRemove)
})
