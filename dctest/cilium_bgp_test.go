package dctest

import (
	_ "embed"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

//go:embed testdata/cilium_bgp.yaml
var bgpYAML []byte

func testCiliumBGPCP() {
	It("should deploy cilium bgp components and load balancer ip pools", func() {
		By("checking cilium-operator and cilium-agent is available")
		checkCiliumOperatorDeployment()
		checkCiliumAgentDaemonSet()

		By("applying BGP related resources")
		stdout, stderr, err := execAtWithInput(bootServers[0], bgpYAML, "kubectl", "apply", "-f", "-")
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
	})
}
