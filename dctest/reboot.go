package dctest

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/cybozu-go/sabakan/v2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

func fetchClusterNodes() (map[string]bool, error) {
	stdout, stderr, err := execAt(bootServers[0], "ckecli", "cluster", "get")
	if err != nil {
		return nil, fmt.Errorf("stdout=%s, stderr=%s err=%v", stdout, stderr, err)
	}

	cluster := new(ckeCluster)
	err = yaml.Unmarshal(stdout, cluster)
	if err != nil {
		return nil, err
	}

	m := make(map[string]bool)
	for _, n := range cluster.Nodes {
		m[n.Address] = n.ControlPlane
	}
	return m, nil
}

// TestRebootAllBootServers tests all boot servers are normal after reboot
func TestRebootAllBootServers() {
	It("runs systemd service on all boot servers after reboot", func() {
		By("rebooting all boot servers")
		for _, host := range bootServers {
			// Exit code is 255 when ssh is disconnected
			execAt(host, "sudo", "reboot")
		}

		By("waiting all boot servers are online")
		err := prepareSSHClients(bootServers...)
		Expect(err).NotTo(HaveOccurred())

		By("checking services on the boot servers are running after reboot")
		checkSystemdServicesOnBoot()

		By("checking sabakan is available")
		Eventually(func() error {
			stdout, stderr, err := execAt(bootServers[0], "sabactl", "machines", "get")
			if err != nil {
				return fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
			}
			return nil
		}).Should(Succeed())
	})
}

// TestRebootAllNodes tests all nodes stop scenario
func TestRebootAllNodes() {
	It("can access a pod from another pod running on different node", func() {
		manifest := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-reboot-test
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: nginx-reboot-test
  template:
    metadata:
      labels:
        app.kubernetes.io/name: nginx-reboot-test
    spec:
      containers:
      - image: quay.io/cybozu/testhttpd:0
        name: testhttpd
        ports:
        - containerPort: 8000
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: debug-reboot-test
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: debug-reboot-test
  template:
    metadata:
      labels:
        app.kubernetes.io/name: debug-reboot-test
    spec:
      securityContext:
        runAsUser: 10000
      containers:
      - image: quay.io/cybozu/ubuntu-debug:18.04
        command:
        - pause
        name: pause
`
		stdout, stderr, err := execAtWithInput(bootServers[0], []byte(manifest), "kubectl", "apply", "-f", "-")
		Expect(err).NotTo(HaveOccurred(), "failed to apply manifests. stdout=%s, stderr=%s", stdout, stderr)
		execSafeAt(bootServers[0], "kubectl", "expose", "deploy", "nginx-reboot-test", "--port=80", "--target-port=8000", "--name=nginx-reboot-test")
		Eventually(func() error {
			debugPodName, stderr, err := execAt(bootServers[0], "kubectl", "get", "pod", "-l=app.kubernetes.io/name=debug-reboot-test", "-o=jsonpath='{.items[0].metadata.name}'")
			if err != nil {
				return fmt.Errorf("failed to get debug-reboot-test. stderr: %s, err: %v", stderr, err)
			}
			stdout, stderr, err := execAt(bootServers[0], "kubectl", "exec", string(debugPodName), "curl", "-s", "http://nginx-reboot-test")
			if err != nil {
				return fmt.Errorf("unable to exec curl. stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
			}
			return nil
		}).Should(Succeed())
	})

	var beforeNodes map[string]bool
	It("fetch cluster nodes", func() {
		var err error
		beforeNodes, err = fetchClusterNodes()
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("stop CKE sabakan integration", func() {
		execSafeAt(bootServers[0], "ckecli", "sabakan", "disable")
	})

	It("reboots all nodes", func() {
		By("getting machines list")
		stdout, _, err := execAt(bootServers[0], "sabactl", "machines", "get")
		Expect(err).ShouldNot(HaveOccurred())
		var machines []sabakan.Machine
		err = json.Unmarshal(stdout, &machines)
		Expect(err).ShouldNot(HaveOccurred())

		By("shutdown all nodes")
		// Skip reboot vm on rack-3 because IPMI is not initialized
		for _, m := range machines {
			if m.Spec.Role == "boot" || m.Spec.Rack == 3 {
				continue
			}
			stdout, stderr, err := execAt(bootServers[0], "neco", "ipmipower", "stop", m.Spec.IPv4[0])
			Expect(err).ShouldNot(HaveOccurred(), "stdout: %s, stderr: %s", stdout, stderr)
		}

		By("wait for rebooting")
		preReboot := make(map[string]bool)
		for _, m := range machines {
			if m.Spec.Role == "boot" {
				continue
			}
			preReboot[m.Spec.IPv4[0]] = true
		}
		Eventually(func() error {
			result, err := getSerfWorkerMembers()
			if err != nil {
				return err
			}
			for _, member := range result.Members {
				addrs := strings.Split(member.Addr, ":")
				if len(addrs) != 2 {
					return fmt.Errorf("unexpected addr: %s", member.Addr)
				}
				addr := addrs[0]
				if preReboot[addr] && member.Status != "alive" {
					delete(preReboot, addr)
				}
			}
			if len(preReboot) > 0 {
				fmt.Println("retry to ipmipower-stop", preReboot)
				for addr := range preReboot {
					stdout, stderr, err := execAt(bootServers[0], "neco", "ipmipower", "stop", addr)
					if err != nil {
						fmt.Println("unable to ipmipower-stop", addr, "stdout:", string(stdout), "stderr:", string(stderr))
					}
				}
				return fmt.Errorf("some nodes are still starting reboot: %v", preReboot)
			}
			return nil
		}).Should(Succeed())

		By("start all nodes")
		//　Wait for each booting to balance the load on the instance
		const waitInterval = 190
		ticker := time.NewTicker(waitInterval * time.Second)
		defer ticker.Stop()
		for _, m := range machines {
			if m.Spec.Rack == 3 {
				continue
			}
			fmt.Println("ipmipower-start", m.Spec.IPv4[0])
			stdout, stderr, err := execAt(bootServers[0], "neco", "ipmipower", "start", m.Spec.IPv4[0])
			Expect(err).ShouldNot(HaveOccurred(), "stdout: %s, stderr: %s", stdout, stderr)
			select {
			case <-ticker.C:
				fmt.Printf("slept %d seconds\n", waitInterval)
			}
		}

		By("wait for recovery of all nodes")
		Eventually(func() error {
			nodes, err := fetchClusterNodes()
			if err != nil {
				return err
			}
			result, err := getSerfWorkerMembers()
			if err != nil {
				return err
			}

			// Debug log
			var serfMember []string
			for _, mem := range result.Members {
				serfMember = append(serfMember, mem.Name+":"+mem.Status)
			}
			sort.Strings(serfMember)
			fmt.Printf("%d: %s\n", len(result.Members), strings.Join(serfMember, ","))

		OUTER:
			for k := range nodes {
				for _, m := range result.Members {
					addrs := strings.Split(m.Addr, ":")
					if len(addrs) != 2 {
						return fmt.Errorf("unexpected addr: %s", m.Addr)
					}
					addr := addrs[0]
					if addr == k {
						if m.Status != "alive" {
							fmt.Println("retry to ipmipower-start", addr)
							stdout, stderr, err := execAt(bootServers[0], "neco", "ipmipower", "start", addr)
							if err != nil {
								fmt.Println("unable to ipmipower-start", addr, "stdout:", string(stdout), "stderr:", string(stderr))
							}
							return fmt.Errorf("reboot failed: %s, %v", k, m)
						}
						continue OUTER
					}
				}
				return fmt.Errorf("cannot find in serf members: %s", k)
			}
			return nil
		}).Should(Succeed())
	})

	It("sets all nodes' machine state to healthy", func() {
		By("getting machine state")
		Eventually(func() error {
			stdout, stderr, err := execAt(bootServers[0], "sabactl", "machines", "get")
			if err != nil {
				return fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
			}

			var machines []sabakan.Machine
			err = json.Unmarshal(stdout, &machines)
			if err != nil {
				return err
			}

			for _, m := range machines {
				if m.Spec.Role == "boot" {
					continue
				}
				if m.Status.State != "healthy" {
					return fmt.Errorf("sabakan machine state of %s is not healthy: %s", m.Spec.Serial, m.Status.State)
				}
			}

			return nil
		}).Should(Succeed())
	})

	It("re-enable CKE sabakan integration", func() {
		execSafeAt(bootServers[0], "ckecli", "sabakan", "enable")
	})

	It("fetch cluster nodes", func() {
		Eventually(func() error {
			afterNodes, err := fetchClusterNodes()
			if err != nil {
				return err
			}

			if !reflect.DeepEqual(beforeNodes, afterNodes) {
				return fmt.Errorf("cluster nodes mismatch after reboot: before=%v after=%v", beforeNodes, afterNodes)
			}

			return nil
		}).Should(Succeed())
	})

	It("can access a pod from another pod running on different node, even after rebooting", func() {
		Eventually(func() error {
			debugPodName, stderr, err := execAt(bootServers[0], "kubectl", "get", "pod", "-l=app.kubernetes.io/name=debug-reboot-test", "-o=jsonpath='{.items[0].metadata.name}'")
			if err != nil {
				return fmt.Errorf("failed to get debug-reboot-test pod. stderr: %s, err: %v", stderr, err)
			}
			stdout, stderr, err := execAt(bootServers[0], "kubectl", "exec", string(debugPodName), "curl", "http://nginx-reboot-test")
			if err != nil {
				return fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
			}
			return nil
		}).Should(Succeed())
	})

	It("can pull new image from internet", func() {
		By("deploying a new pod whose image-pull-policy is Always")
		testPod := "test-new-image"
		execSafeAt(bootServers[0], "kubectl", "run", testPod,
			"--image=quay.io/cybozu/testhttpd:0", "--image-pull-policy=Always", "--generator=run-pod/v1")
		By("checking the pod is running")
		Eventually(func() error {
			stdout, stderr, err := execAt(bootServers[0], "kubectl", "get", "pod", testPod, "-o=json")
			if err != nil {
				return fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
			}
			pod := new(corev1.Pod)
			err = json.Unmarshal(stdout, &pod)
			if err != nil {
				return err
			}
			if pod.Status.Phase != "Running" {
				return fmt.Errorf("%s is not Running. current status: %v", testPod, pod.Status)
			}
			return nil
		}).Should(Succeed())

		By("deleting " + testPod)
		execSafeAt(bootServers[0], "kubectl", "delete", "pod", testPod)
	})
}
