package dctest

import (
	. "github.com/onsi/ginkgo/v2"
)

func testTools() {
	It("should install necoip command", func() {
		nodeName := execSafeAt(bootServers[0], "kubectl", "get", "node", "-o", "jsonpath='{.items[0].metadata.name}'")
		execSafeAt(bootServers[0], "necoip", string(nodeName))
	})

	It("should install nsdump command", func() {
		execSafeAt(bootServers[0], "nsdump", "default")
	})

	It("should install clusterdump command", func() {
		execSafeAt(bootServers[0], "clusterdump")
	})
}
