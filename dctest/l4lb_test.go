package dctest

import (
	"bytes"
	_ "embed"
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
		Eventually(func(g Gomega) {
			checkDeploymentReplicas(g, "testhttpd", ns, 2)
		}).Should(Succeed())
	})

	It("should distribute incoming traffic across backend Pods", func() {
		By("waiting service are ready")
		var targetIP, targetIPForLocal string
		Eventually(func(g Gomega) {
			service := kubectlGetSafe[corev1.Service](g, "-n", ns, "service/testhttpd")
			g.Expect(len(service.Status.LoadBalancer.Ingress)).NotTo(Equal(0), "LoadBalancer status is not updated")
			targetIP = service.Status.LoadBalancer.Ingress[0].IP

			service = kubectlGetSafe[corev1.Service](g, "-n", ns, "service/testhttpd-local")
			g.Expect(len(service.Status.LoadBalancer.Ingress)).NotTo(Equal(0), "LoadBalancer status is not updated")
			targetIPForLocal = service.Status.LoadBalancer.Ingress[0].IP
		}).Should(Succeed())

		By("access service from external")
		Eventually(func(g Gomega) {
			err := exec.Command("ip", "netns", "exec", "external", "curl", targetIP, "-m", "5").Run()
			g.Expect(err).NotTo(HaveOccurred())

			err = exec.Command("ip", "netns", "exec", "external", "curl", targetIPForLocal, "-m", "5").Run()
			g.Expect(err).NotTo(HaveOccurred())
		}).Should(Succeed())

		By("access service from external(Inbound packets have the tos)")
		Expect(exec.Command("ip", "netns", "exec", "external",
			"iptables", "-t", "mangle", "-A", "OUTPUT", "-p", "TCP", "--dport", "80", "-j", "TOS", "--set-tos", "0x20").Run()).ShouldNot(HaveOccurred())
		Eventually(func(g Gomega) {
			err := exec.Command("ip", "netns", "exec", "external", "curl", targetIP, "-m", "5").Run()
			g.Expect(err).NotTo(HaveOccurred())

			err = exec.Command("ip", "netns", "exec", "external", "curl", targetIPForLocal, "-m", "5").Run()
			g.Expect(err).NotTo(HaveOccurred())
		}).Should(Succeed())
		Expect(exec.Command("ip", "netns", "exec", "external",
			"iptables", "-t", "mangle", "-D", "OUTPUT", "-p", "TCP", "--dport", "80", "-j", "TOS", "--set-tos", "0x20").Run()).ShouldNot(HaveOccurred())

		By("prepare a test client")
		podList := kubectlGetSafe[corev1.PodList](Default, "pods", "-n", ns, "-l", "app.kubernetes.io/name=testhttpd")
		scheduledNodeSet := make(map[string]struct{})
		for _, pod := range podList.Items {
			scheduledNodeSet[pod.Spec.NodeName] = struct{}{}
		}

		stdout, stderr, err := execAt(bootServers[0], "ckecli", "cluster", "get")
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
		Eventually(func(g Gomega) {
			podList := kubectlGetSafe[corev1.PodList](g, "pods", "-n", ns, "-l", "app.kubernetes.io/name=ubuntu-l4lb-client")
			g.Expect(len(podList.Items)).To(Equal(1))
			podName := podList.Items[0].Name

			stdout, stderr, err = execAt(bootServers[0], "kubectl", "exec", "-n", ns, podName, "--", "curl", targetIP, "-m", "5")
			g.Expect(err).NotTo(HaveOccurred(), "stdout: %s, stderr: %s, err: %v", stdout, stderr, err)

			stdout, stderr, err = execAt(bootServers[0], "kubectl", "exec", "-n", ns, podName, "--", "curl", targetIPForLocal, "-m", "5")
			g.Expect(err).NotTo(HaveOccurred(), "stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
		}).Should(Succeed())

		By("access service from boot-0 (via cke-localproxy)")
		Eventually(func(g Gomega) {
			stdout, stderr, err := execAt(bootServers[0], "curl", "http://testhttpd.testl4lb.svc/", "-m", "5")
			g.Expect(err).NotTo(HaveOccurred(), "stdout: %s, stderr: %s, err: %v", stdout, stderr, err)

			stdout, stderr, err = execAt(bootServers[0], "curl", "http://testhttpd-local.testl4lb.svc/", "-m", "5")
			g.Expect(err).NotTo(HaveOccurred(), "stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
		}).Should(Succeed())

		By("removing the testl4lb namespace")
		execSafeAt(bootServers[0], "kubectl", "delete", "ns", ns)
	})
}

func checkDeploymentReplicas(g Gomega, name, namespace string, desiredReplicas int) {
	deployment := kubectlGetSafe[appsv1.Deployment](g, "deployment", "-n", namespace, name)
	if desiredReplicas < 0 {
		if deployment.Spec.Replicas == nil {
			desiredReplicas = 1
		} else {
			desiredReplicas = int(*deployment.Spec.Replicas)
		}
	}

	g.Expect(int(deployment.Status.AvailableReplicas)).To(Equal(desiredReplicas),
		"AvailableReplicas of Deployment %s/%s is not %d: %d", namespace, name, desiredReplicas, deployment.Status.AvailableReplicas)

	g.Expect(int(deployment.Status.UpdatedReplicas)).To(Equal(desiredReplicas),
		"UpdatedReplicas of Deployment %s/%s is not %d: %d", namespace, name, desiredReplicas, deployment.Status.UpdatedReplicas)
}
