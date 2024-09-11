package dctest

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/cybozu-go/neco"
	necorebooter "github.com/cybozu-go/neco/pkg/neco-rebooter"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/yaml"
)

// testRebootAllBootServers tests all boot servers are normal after reboot
func testRebootAllBootServers() {
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

		// .kube directory is mounted by tmpfs, so reissuing config file is necessary
		By("generating kubeconfig for cluster admin")
		Eventually(func() error {
			_, stderr, err := execAt(bootServers[0], "ckecli", "kubernetes", "issue", ">", ".kube/config")
			if err != nil {
				return fmt.Errorf("err: %v, stderr: %s", err, stderr)
			}
			return nil
		}).Should(Succeed())
	})
}

func testCKERebootGracefully() {
	It("can reboot all workers gracefully", func() {
		By("generating kubeconfig for cluster admin")
		Eventually(func() error {
			_, stderr, err := execAt(bootServers[0], "ckecli", "kubernetes", "issue", ">", ".kube/config")
			if err != nil {
				return fmt.Errorf("err: %v, stderr: %s", err, stderr)
			}
			return nil
		}).Should(Succeed())

		workersBefore, err := getSerfWorkerMembers()
		Expect(err).NotTo(HaveOccurred())

		By("adding all nodes to CKE reboot-queue")
		var nodeList corev1.NodeList
		nodeSet := map[string]bool{}
		for _, node := range nodeList.Items {
			nodeSet[node.Name] = true
		}
		nodeListJson := execSafeAt(bootServers[0], "kubectl", "get", "node", "-ojson")
		err = json.Unmarshal(nodeListJson, &nodeList)
		Expect(err).NotTo(HaveOccurred())
		execSafeAt(bootServers[0], "ckecli", "rq", "enable")
		for _, node := range nodeList.Items {
			_, stderr, err := execAtWithInput(bootServers[0], []byte(strings.Split(node.Name, ":")[0]), "ckecli", "rq", "add", "-")
			Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)
		}

		By("waiting for reboot-queue to be processed")
		Eventually(func() error {
			stdout, stderr, err := execAt(bootServers[0], "ckecli", "rq", "list")
			if err != nil {
				return fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
			}
			if strings.TrimSpace(string(stdout)) != "null" {
				return fmt.Errorf("reboot-queue is not processed")
			}
			return nil
		}, 50*time.Minute).Should(Succeed())

		workersAfter, err := getSerfWorkerMembers()
		Expect(err).NotTo(HaveOccurred())
		for _, before := range workersBefore.Members {
			for _, after := range workersAfter.Members {
				if before.Name == after.Name && nodeSet[strings.Split(before.Addr, ":")[0]] {
					Expect(after.Tags["uptime"]).NotTo(Equal(before.Tags["uptime"]))
				}
			}
		}
	})
}

func testNecoRebooterRebootGracefully() {
	It("can enqueue workers to reboot queue correctly", func() {
		By("generating kubeconfig for cluster admin")
		Eventually(func() error {
			_, stderr, err := execAt(bootServers[0], "ckecli", "kubernetes", "issue", ">", ".kube/config")
			if err != nil {
				return fmt.Errorf("err: %v, stderr: %s", err, stderr)
			}
			return nil
		}).Should(Succeed())

		roles := []string{"", "cs", "ss"}

		execSafeAt(bootServers[0], "ckecli", "rq", "disable")

		for _, role := range roles {
			kubectlCommand := []string{"kubectl", "get", "node", "-ojson"}
			necoRebootWorkerOptions := ""
			if role != "" {
				kubectlCommand = append(kubectlCommand, "-lcke.cybozu.com/role="+role)
				necoRebootWorkerOptions = "--role=" + role
				By("trying to reboot " + role + " nodes")
			} else {
				By("trying to reboot all nodes")
			}

			var nodeList corev1.NodeList
			type Node struct {
				rack string
				role string
			}
			nodeSet := map[string]Node{}
			nodeListJson := execSafeAt(bootServers[0], kubectlCommand...)
			err := json.Unmarshal(nodeListJson, &nodeList)
			Expect(err).NotTo(HaveOccurred())
			for _, node := range nodeList.Items {
				rack := node.Labels["topology.kubernetes.io/zone"]
				Expect(rack).NotTo(BeEmpty())
				nodeRole := node.Labels["cke.cybozu.com/role"]
				Expect(nodeRole).NotTo(BeEmpty())
				nodeSet[node.Name] = Node{
					rack: rack,
					role: nodeRole,
				}
			}

			By("Adding reboot-list entry for " + role + " nodes")
			execSafeAt(bootServers[0], "yes", "|", "neco", "rebooter", "reboot-worker "+necoRebootWorkerOptions)
			rle := []neco.RebootListEntry{}
			rleJson := execSafeAt(bootServers[0], "neco", "rebooter", "list")
			Expect(strings.TrimSpace(string(rleJson))).NotTo(Equal("null"))
			err = json.Unmarshal(rleJson, &rle)
			Expect(err).NotTo(HaveOccurred())
			// Every target kubernetes nodes are pushed to reboot list exactly once.
			for _, e := range rle {
				if e.Status == neco.RebootListEntryStatusCancelled {
					continue
				}
				rack := nodeSet[e.Node].rack
				Expect(e.Group).To(Equal(rack))
				nodeRole := nodeSet[e.Node].role
				Expect(e.RebootTime).To(Equal(nodeRole))
				Expect(nodeSet).To(HaveKey(e.Node))
				delete(nodeSet, e.Node)
			}
			Expect(nodeSet).To(BeEmpty())

			By("Cancelling all reboot-list entries")
			execSafeAt(bootServers[0], "neco", "rebooter", "cancel-all")
		}

		By("waiting for all reboot-list removed")
		Eventually(func() error {
			stdout, stderr, err := execAt(bootServers[0], "neco", "rebooter", "list")

			if err != nil {
				return fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
			}
			if strings.TrimSpace(string(stdout)) != "null" {
				return fmt.Errorf("reboot-list is not processed")
			}
			return nil
		}, 30*time.Minute).Should(Succeed())

		By("waiting for reboot-queue removed")
		Eventually(func() error {
			stdout, stderr, err := execAt(bootServers[0], "ckecli", "rq", "list")

			if err != nil {
				return fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
			}
			if strings.TrimSpace(string(stdout)) != "null" {
				return fmt.Errorf("reboot-queue is not processed")
			}
			return nil
		}, 30*time.Minute).Should(Succeed())

		By("waiting for all nodes up")
		Eventually(func() error {
			workers, err := getSerfWorkerMembers()
			if err != nil {
				return err
			}
			for _, worker := range workers.Members {
				if worker.Status != "alive" {
					return fmt.Errorf("worker %s is not alive", worker.Name)
				}
			}
			return nil
		}).Should(Succeed())
	})

	It("can move group when reboot-queue is stuck", func() {
		By("generating kubeconfig for cluster admin")
		Eventually(func() error {
			_, stderr, err := execAt(bootServers[0], "ckecli", "kubernetes", "issue", ">", ".kube/config")
			if err != nil {
				return fmt.Errorf("err: %v, stderr: %s", err, stderr)
			}
			return nil
		}).Should(Succeed())

		By("getting available racks")
		stdout, stderr, err := execAt(bootServers[0], "kubectl", "get", "nodes", "-o", "json")
		Expect(err).NotTo(HaveOccurred(), "stdout: %s, stderr: %s", stdout, stderr)
		nodes := corev1.NodeList{}
		err = json.Unmarshal(stdout, &nodes)
		Expect(err).NotTo(HaveOccurred())
		racks := []string{}
		for _, node := range nodes.Items {
			rack := node.Labels["topology.kubernetes.io/zone"]
			if len(rack) == 0 {
				continue
			}
			racks = append(racks, rack)
		}
		slices.Sort(racks)
		racks = slices.Compact(racks)
		Expect(len(racks)).To(BeNumerically(">", 1))

		By("changing neco-rebooter config")
		config := necorebooter.Config{
			RebootTimes: []necorebooter.RebootTimes{
				{
					Name: "cs",
					LabelSelector: necorebooter.LabelSelector{
						MatchLabels: map[string]string{
							"cke.cybozu.com/role": "cs",
						},
					},
					Times: necorebooter.Times{
						Allow: []string{"* * * * *"},
					},
				},
				{
					Name: "ss",
					LabelSelector: necorebooter.LabelSelector{
						MatchLabels: map[string]string{
							"cke.cybozu.com/role": "ss",
						},
					},
					Times: necorebooter.Times{
						Allow: []string{"* * * * *"},
					},
				},
			},
			GroupLabelKey: "topology.kubernetes.io/zone",
		}
		configYaml, err := yaml.Marshal(config)
		Expect(err).NotTo(HaveOccurred())
		for _, boot := range bootServers {
			stdout, stderr, err := execAtWithInput(boot, configYaml, "sudo", "tee", "/usr/share/neco/neco-rebooter.yaml")
			Expect(err).NotTo(HaveOccurred(), "stdout: %s, stderr: %s", stdout, stderr)
			execSafeAt(boot, "sudo", "systemctl", "restart", "neco-rebooter")
		}

		pod := corev1.Pod{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "default",
				Labels: map[string]string{
					"app": "test-pod",
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:    "test-container",
						Image:   "ghcr.io/cybozu/ubuntu:22.04",
						Command: []string{"pause"},
					},
				},
				NodeSelector: map[string]string{
					"topology.kubernetes.io/zone": racks[0],
				},
			},
		}
		pdb := policyv1.PodDisruptionBudget{
			TypeMeta: metav1.TypeMeta{
				Kind:       "PodDisruptionBudget",
				APIVersion: "policy/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pdb",
				Namespace: "default",
			},
			Spec: policyv1.PodDisruptionBudgetSpec{
				MaxUnavailable: &intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: 0,
				},
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "test-pod",
					},
				},
			},
		}
		podYaml, err := json.Marshal(pod)
		Expect(err).NotTo(HaveOccurred())
		pdbYaml, err := json.Marshal(pdb)
		Expect(err).NotTo(HaveOccurred())

		By("creating a pod")
		stdout, stderr, err = execAtWithInput(bootServers[0], podYaml, "kubectl", "apply", "-f", "-")
		Expect(err).NotTo(HaveOccurred(), "stdout: %s, stderr: %s", stdout, stderr)
		By("creating a pdb")
		stdout, stderr, err = execAtWithInput(bootServers[0], pdbYaml, "kubectl", "apply", "-f", "-")
		Expect(err).NotTo(HaveOccurred(), "stdout: %s, stderr: %s", stdout, stderr)

		By(fmt.Sprintf("adding %s nodes to reboot-list", racks[0]))
		// [:4] is number of "rack"
		execSafeAt(bootServers[0], "yes", "|", "neco", "rebooter", "reboot-worker", "--rack="+racks[0][4:])

		By("enable rebooting")
		execSafeAt(bootServers[0], "ckecli", "rq", "enable")
		execSafeAt(bootServers[0], "neco", "rebooter", "enable")

		By("waiting for reboot-list to be enqueued")
		Eventually(func() error {
			stdout, stderr, err := execAt(bootServers[0], "ckecli", "rq", "list")
			if err != nil {
				return fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
			}
			if strings.TrimSpace(string(stdout)) == "null" {
				return fmt.Errorf("reboot-queue is not processed")
			}
			return nil
		}).Should(Succeed())

		By(fmt.Sprintf("adding %s nodes to reboot-list", racks[1]))
		// [:4] is number of "rack"
		execSafeAt(bootServers[0], "yes", "|", "neco", "rebooter", "reboot-worker", "--rack="+racks[1][4:])

		By(fmt.Sprintf("waiting for skipping %s and moving to %s", racks[0], racks[1]))
		Eventually(func() error {
			stdout, stderr, err := execAt(bootServers[0], "neco", "rebooter", "show-processing-group")
			if err != nil {
				return fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
			}
			if strings.TrimSpace(string(stdout)) != racks[1] {
				return fmt.Errorf("reboot-queue is not processed")
			}
			return nil
		}).Should(Succeed())

		By("canceling all reboot-list entries")
		execSafeAt(bootServers[0], "neco", "rebooter", "cancel-all")

		By("delete the pod and pdb")
		execSafeAt(bootServers[0], "kubectl", "delete", "pod", "test-pod")
		execSafeAt(bootServers[0], "kubectl", "delete", "pdb", "test-pdb")

		By("waiting for all reboot-list removed")
		Eventually(func() error {
			stdout, stderr, err := execAt(bootServers[0], "neco", "rebooter", "list")

			if err != nil {
				return fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
			}
			if strings.TrimSpace(string(stdout)) != "null" {
				return fmt.Errorf("reboot-list is not processed")
			}
			return nil
		}, 30*time.Minute).Should(Succeed())

		By("waiting for reboot-queue removed")
		Eventually(func() error {
			stdout, stderr, err := execAt(bootServers[0], "ckecli", "rq", "list")

			if err != nil {
				return fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
			}
			if strings.TrimSpace(string(stdout)) != "null" {
				return fmt.Errorf("reboot-queue is not processed")
			}
			return nil
		}, 30*time.Minute).Should(Succeed())
	})

	It("can reboot all workers gracefully", func() {
		By("generating kubeconfig for cluster admin")
		Eventually(func() error {
			_, stderr, err := execAt(bootServers[0], "ckecli", "kubernetes", "issue", ">", ".kube/config")
			if err != nil {
				return fmt.Errorf("err: %v, stderr: %s", err, stderr)
			}
			return nil
		}).Should(Succeed())

		workersBefore, err := getSerfWorkerMembers()
		Expect(err).NotTo(HaveOccurred())

		By("rebooting all workers")
		execSafeAt(bootServers[0], "ckecli", "rq", "enable")
		execSafeAt(bootServers[0], "neco", "rebooter", "enable")
		execSafeAt(bootServers[0], "yes", "|", "neco", "rebooter", "reboot-worker")

		By("waiting for reboot-list to be processed")
		Eventually(func() error {
			stdout, stderr, err := execAt(bootServers[0], "neco", "rebooter", "list")
			if err != nil {
				return fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
			}
			if strings.TrimSpace(string(stdout)) != "null" {
				return fmt.Errorf("reboot-queue is not processed")
			}
			return nil
		}, 90*time.Minute).Should(Succeed())

		workersAfter, err := getSerfWorkerMembers()
		Expect(err).NotTo(HaveOccurred())
		for _, before := range workersBefore.Members {
			for _, after := range workersAfter.Members {
				if before.Name == after.Name {
					Expect(after.Tags["uptime"]).NotTo(Equal(before.Tags["uptime"]))
				}
			}
		}
	})
}
