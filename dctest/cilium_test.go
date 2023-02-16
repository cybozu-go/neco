package dctest

import (
	"encoding/json"
	"errors"
	"fmt"

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
	EventuallyWithOffset(1, func() error {
		stdout, _, err := execAt(bootServers[0], "kubectl", "--namespace=kube-system",
			"get", "daemonsets/cilium", "-o=json")
		if err != nil {
			return err
		}

		daemonset := new(appsv1.DaemonSet)
		err = json.Unmarshal(stdout, daemonset)
		if err != nil {
			return err
		}

		if daemonset.Status.NumberReady != daemonset.Status.DesiredNumberScheduled {
			return fmt.Errorf("NumberReady: %d, DesiredNumberScheduled: %d", daemonset.Status.NumberReady, daemonset.Status.DesiredNumberScheduled)
		}
		if daemonset.Status.NumberReady == 0 {
			return errors.New("NumberReady == 0")
		}
		return nil
	}).Should(Succeed())
}

func checkCiliumOperatorDeployment() {
	EventuallyWithOffset(1, func() error {
		stdout, _, err := execAt(bootServers[0], "kubectl", "--namespace=kube-system",
			"get", "deployment/cilium-operator", "-o=json")
		if err != nil {
			return err
		}

		deployment := new(appsv1.Deployment)
		err = json.Unmarshal(stdout, deployment)
		if err != nil {
			return err
		}

		if deployment.Status.ReadyReplicas != 2 {
			return fmt.Errorf("ReadyReplicas is not 2 but %d", deployment.Status.ReadyReplicas)
		}
		return nil
	}).Should(Succeed())
}
func checkHubbleRelayDeployment() {
	EventuallyWithOffset(1, func() error {
		stdout, _, err := execAt(bootServers[0], "kubectl", "--namespace=kube-system",
			"get", "deployment/hubble-relay", "-o=json")
		if err != nil {
			return err
		}

		deployment := new(appsv1.Deployment)
		err = json.Unmarshal(stdout, deployment)
		if err != nil {
			return err
		}

		if deployment.Status.ReadyReplicas != 1 {
			return fmt.Errorf("ReadyReplicas is not 1 but %d", deployment.Status.ReadyReplicas)
		}

		stdout, _, err = execAt(bootServers[0], "hubble", "status", "-o", "json", "--server",
			"hubble-relay.kube-system.svc:443", "--tls", "--tls-allow-insecure")
		if err != nil {
			return err
		}

		hubbleStatus := new(hubbleStatus)
		err = json.Unmarshal(stdout, hubbleStatus)
		if err != nil {
			return err
		}

		if hubbleStatus.NumUnavailableNodes != 0 {
			return fmt.Errorf("NumUnavailableNodes is not 0 but %d", hubbleStatus.NumUnavailableNodes)
		}

		return nil
	}).Should(Succeed())
}
