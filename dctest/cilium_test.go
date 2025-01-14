package dctest

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
)

type hubbleStatus struct {
	NumFlows            string `json:"num_flows"`
	MaxFlows            string `json:"max_flows"`
	SeenFlows           string `json:"seen_flows"`
	UptimeNs            string `json:"uptime_ns"`
	NumConnectedNodes   int    `json:"num_connected_nodes"`
	NumUnavailableNodes int    `json:"num_unavailable_nodes"`
	Version             string `json:"version"`
}

// testCiliumSetup tests Cilium setup
func testCilium() {
	It("should be deployed successfully", func() {
		By("waiting for cilium DaemonSet and cilium-operator Deployment")
		checkCiliumAgentDaemonSet()
		checkCiliumOperatorDeployment()
		checkHubbleRelayDeployment()
	})
}

func checkCiliumAgentDaemonSet() {
	EventuallyWithOffset(1, func(g Gomega) {
		ds := kubectlGetSafe[appsv1.DaemonSet](g, "--namespace=kube-system", "daemonsets/cilium")

		g.Expect(ds.Status.NumberReady).To(BeNumerically(">", 0), "NumberReady should be positive")
		g.Expect(ds.Status.NumberReady).To(Equal(ds.Status.DesiredNumberScheduled),
			"NumberReady: %d, DesiredNumberScheduled: %d", ds.Status.NumberReady, ds.Status.DesiredNumberScheduled)
	}).Should(Succeed())
}

func checkCiliumOperatorDeployment() {
	EventuallyWithOffset(1, func(g Gomega) {
		deployment := kubectlGetSafe[appsv1.Deployment](g, "--namespace=kube-system", "deployment/cilium-operator")
		g.Expect(deployment.Status.ReadyReplicas).To(Equal(int32(2)),
			"ReadyReplicas of cilium-operator should be 2 but %d", deployment.Status.ReadyReplicas)
	}).Should(Succeed())
}

func checkHubbleRelayDeployment() {
	EventuallyWithOffset(1, func(g Gomega) {
		deployment := kubectlGetSafe[appsv1.Deployment](g, "--namespace=kube-system", "deployment/hubble-relay")
		g.Expect(deployment.Status.ReadyReplicas).To(Equal(int32(1)),
			"ReadyReplicas of hubble-relay should be 1 but %d", deployment.Status.ReadyReplicas)

		hubbleStatus := unmarshalSafe[hubbleStatus](g, execSafeGomegaAt(g, bootServers[0], "hubble", "status", "-o", "json",
			"--server", "hubble-relay.kube-system.svc:443", "--tls", "--tls-allow-insecure"))
		g.Expect(hubbleStatus.NumUnavailableNodes).To(Equal(0),
			"NumUnavailableNodes should be 0 but %d", hubbleStatus.NumUnavailableNodes)
	}).Should(Succeed())
}
