package dctest

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"text/template"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

//go:embed testdata/l4lb.yaml
var l4lbYAML []byte

//go:embed testdata/l4lb_client.yaml
var l4lbClientYAML string

func testL4LB() {
	ns := "testl4lb"
	It("should deploy load balancer type service", func() {
		By("creating deployments and service")
		stdout, stderr, err := execAtWithInput(bootServers[0], l4lbYAML, "kubectl", "apply", "-f", "-")
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		By("waiting testhttpd pods are ready")
		Eventually(func() error {
			return checkDeploymentReplicas("testhttpd", ns, 2)
		}).Should(Succeed())
	})

	It("should distribute incoming traffic across backend Pods", func() {
		By("waiting service are ready")
		var targetIP, targetIPForLocal string
		Eventually(func() error {
			stdout, stderr, err := execAt(bootServers[0], "kubectl", "-n", ns, "get", "service/testhttpd", "-o", "json")
			if err != nil {
				return fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
			}
			service := new(corev1.Service)
			err = json.Unmarshal(stdout, service)
			if err != nil {
				return err
			}
			if len(service.Status.LoadBalancer.Ingress) == 0 {
				return errors.New("LoadBalancer status is not updated")
			}
			targetIP = service.Status.LoadBalancer.Ingress[0].IP

			stdout, stderr, err = execAt(bootServers[0], "kubectl", "-n", ns, "get", "service/testhttpd-local", "-o", "json")
			if err != nil {
				return fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
			}
			service = new(corev1.Service)
			err = json.Unmarshal(stdout, service)
			if err != nil {
				return err
			}
			if len(service.Status.LoadBalancer.Ingress) == 0 {
				return errors.New("LoadBalancer status is not updated")
			}
			targetIPForLocal = service.Status.LoadBalancer.Ingress[0].IP
			return nil
		}).Should(Succeed())

		By("access service from external")
		Eventually(func() error {
			err := exec.Command("ip", "netns", "exec", "external", "curl", targetIP, "-m", "5").Run()
			if err != nil {
				return err
			}

			return exec.Command("ip", "netns", "exec", "external", "curl", targetIPForLocal, "-m", "5").Run()
		}).Should(Succeed())
		Consistently(func() error {
			err := exec.Command("ip", "netns", "exec", "external", "curl", targetIP, "-m", "5").Run()
			if err != nil {
				return err
			}

			return exec.Command("ip", "netns", "exec", "external", "curl", targetIPForLocal, "-m", "5").Run()
		}).Should(Succeed())

		By("access service from external(Inbound packets have the tos)")
		Expect(exec.Command("ip", "netns", "exec", "external",
			"iptables", "-t", "mangle", "-A", "OUTPUT", "-p", "TCP", "--dport", "80", "-j", "TOS", "--set-tos", "0x20").Run()).ShouldNot(HaveOccurred())
		Eventually(func() error {
			err := exec.Command("ip", "netns", "exec", "external", "curl", targetIP, "-m", "5").Run()
			if err != nil {
				return err
			}

			return exec.Command("ip", "netns", "exec", "external", "curl", targetIPForLocal, "-m", "5").Run()
		}).Should(Succeed())
		Consistently(func() error {
			err := exec.Command("ip", "netns", "exec", "external", "curl", targetIP, "-m", "5").Run()
			if err != nil {
				return err
			}

			return exec.Command("ip", "netns", "exec", "external", "curl", targetIPForLocal, "-m", "5").Run()
		}).Should(Succeed())
		Expect(exec.Command("ip", "netns", "exec", "external",
			"iptables", "-t", "mangle", "-D", "OUTPUT", "-p", "TCP", "--dport", "80", "-j", "TOS", "--set-tos", "0x20").Run()).ShouldNot(HaveOccurred())

		By("prepare a test client")
		stdout, stderr, err := execAt(bootServers[0], "kubectl", "-n", ns, "get", "pods", "-l", "app.kubernetes.io/name=testhttpd", "-o", "json")
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
		podList := &corev1.PodList{}
		Expect(json.Unmarshal(stdout, podList)).ShouldNot(HaveOccurred())
		scheduledNodeSet := make(map[string]struct{})
		for _, pod := range podList.Items {
			scheduledNodeSet[pod.Spec.NodeName] = struct{}{}
		}

		stdout, stderr, err = execAt(bootServers[0], "ckecli", "cluster", "get")
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
		cluster := new(ckeCluster)
		err = yaml.Unmarshal(stdout, cluster)
		Expect(err).ShouldNot(HaveOccurred(), "data=%s", stdout)
		var selectedNode string
		for _, node := range cluster.Nodes {
			if node.Labels["cke.cybozu.com/role"] != "cs" {
				continue
			}
			if _, ok := scheduledNodeSet[node.Address]; !ok {
				selectedNode = node.Address
				break
			}
		}
		Expect(selectedNode).NotTo(BeEmpty())

		tmpl := template.Must(template.New("").Parse(l4lbClientYAML))
		type tmplParams struct {
			Node string
		}
		buf := new(bytes.Buffer)
		err = tmpl.Execute(buf, tmplParams{selectedNode})
		Expect(err).NotTo(HaveOccurred())
		_, stderr, err = execAtWithInput(bootServers[0], buf.Bytes(), "kubectl", "create", "-f", "-")
		Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)

		By("access service from a Pod")
		stdout, stderr, err = execAt(bootServers[0], "kubectl", "-n", ns, "get", "pods", "-l", "app.kubernetes.io/name=ubuntu-l4lb-client", "-o", "json")
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
		podList = &corev1.PodList{}
		err = json.Unmarshal(stdout, podList)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(len(podList.Items)).To(Equal(1))
		podName := podList.Items[0].Name

		Eventually(func() error {
			stdout, stderr, err = execAt(bootServers[0], "kubectl", "exec", "-n", ns, podName, "--", "curl", targetIP, "-m", "5")
			if err != nil {
				return fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
			}
			stdout, stderr, err = execAt(bootServers[0], "kubectl", "exec", "-n", ns, podName, "--", "curl", targetIPForLocal, "-m", "5")
			if err != nil {
				return fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
			}
			return nil
		}).Should(Succeed())
		Consistently(func() error {
			stdout, stderr, err = execAt(bootServers[0], "kubectl", "exec", "-n", ns, podName, "--", "curl", targetIP, "-m", "5")
			if err != nil {
				return fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
			}
			stdout, stderr, err = execAt(bootServers[0], "kubectl", "exec", "-n", ns, podName, "--", "curl", targetIPForLocal, "-m", "5")
			if err != nil {
				return fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
			}
			return nil
		}).Should(Succeed())

		By("access service from boot-0 (via cke-localproxy)")
		Eventually(func() error {
			stdout, stderr, err := execAt(bootServers[0], "curl", "http://testhttpd.testl4lb.svc/", "-m", "5")
			if err != nil {
				return fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
			}

			stdout, stderr, err = execAt(bootServers[0], "curl", "http://testhttpd-local.testl4lb.svc/", "-m", "5")
			if err != nil {
				return fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
			}
			return nil
		}).Should(Succeed())

		By("removing the testl4lb namespace")
		execSafeAt(bootServers[0], "kubectl", "delete", "ns", ns)
	})
}

func checkDeploymentReplicas(name, namespace string, desiredReplicas int) error {
	stdout, stderr, err := execAt(bootServers[0], "kubectl", "get", "deployment", "-n", namespace, name, "-o", "json")
	if err != nil {
		return fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
	}

	deployment := new(appsv1.Deployment)
	err = json.Unmarshal(stdout, deployment)
	if err != nil {
		return err
	}

	if desiredReplicas < 0 {
		if deployment.Spec.Replicas == nil {
			desiredReplicas = 1
		} else {
			desiredReplicas = int(*deployment.Spec.Replicas)
		}
	}

	if int(deployment.Status.AvailableReplicas) != desiredReplicas {
		return fmt.Errorf("AvailableReplicas of Deployment %s/%s is not %d: %d", namespace, name, desiredReplicas, deployment.Status.AvailableReplicas)
	}
	if int(deployment.Status.UpdatedReplicas) != desiredReplicas {
		return fmt.Errorf("UpdatedReplicas of Deployment %s/%s is not %d: %d", namespace, name, desiredReplicas, deployment.Status.UpdatedReplicas)
	}

	return nil
}
