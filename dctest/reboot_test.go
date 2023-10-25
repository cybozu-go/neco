package dctest

import (
	"encoding/json"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
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

// rebootQueueEntry is part of cke.RebootQueueEntry in github.com/cybozu-go/cke
type rebootQueueEntry struct {
	Node   string `json:"node"`
	Status string `json:"status"`
}

// testRebootGracefully tests graceful reboot of workers
func testRebootGracefully() {
	It("can enqueue workers to reboot queue correctly", func() {
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
			nodeSet := map[string]bool{}
			nodeListJson := execSafeAt(bootServers[0], kubectlCommand...)
			err := json.Unmarshal(nodeListJson, &nodeList)
			Expect(err).NotTo(HaveOccurred())
			for _, node := range nodeList.Items {
				nodeSet[node.Name] = true
			}

			execSafeAt(bootServers[0], "sh", "-c", "yes | neco reboot-worker "+necoRebootWorkerOptions)
			rqe := []rebootQueueEntry{}
			rqeJson := execSafeAt(bootServers[0], "ckecli", "rq", "list")
			Expect(string(rqeJson)).NotTo(Equal("null"))
			err = json.Unmarshal(rqeJson, &rqe)
			Expect(err).NotTo(HaveOccurred())
			// Every target kubernetes nodes are pushed to reboot queue exactly once.
			for _, e := range rqe {
				if e.Status == "cancelled" {
					continue
				}
				Expect(nodeSet).To(HaveKey(e.Node))
				delete(nodeSet, e.Node)
			}
			Expect(nodeSet).To(BeEmpty())

			execSafeAt(bootServers[0], "ckecli", "rq", "cancel-all")
		}

		// make CKE dequeue cancelled entries
		execSafeAt(bootServers[0], "ckecli", "rq", "enable")
		Eventually(func() error {
			stdout, stderr, err := execAt(bootServers[0], "ckecli", "rq", "list")
			if err != nil {
				return fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
			}
			if string(stdout) != "null\n" {
				return fmt.Errorf("reboot-queue is not processed")
			}
			return nil
		}, 30*time.Minute).Should(Succeed())
	})

	It("can reboot all workers gracefully", func() {
		workersBefore, err := getSerfWorkerMembers()
		Expect(err).NotTo(HaveOccurred())

		execSafeAt(bootServers[0], "sh", "-c", "yes | neco reboot-worker")
		Eventually(func() error {
			stdout, stderr, err := execAt(bootServers[0], "ckecli", "rq", "list")
			if err != nil {
				return fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
			}
			if string(stdout) != "null\n" {
				return fmt.Errorf("reboot-queue is not processed")
			}
			return nil
		}, 50*time.Minute).Should(Succeed())

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
