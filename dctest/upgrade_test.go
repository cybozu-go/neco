package dctest

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/sabakan/v3"
	"github.com/hashicorp/go-version"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

// ckeCluster is part of cke.Cluster in github.com/cybozu-go/cke
type ckeCluster struct {
	Nodes []*ckeNode `yaml:"nodes"`
}

// ckeNode is part of cke.Node in github.com/cybozu-go/cke
type ckeNode struct {
	Address      string            `yaml:"address"`
	ControlPlane bool              `yaml:"control_plane"`
	Labels       map[string]string `yaml:"labels"`
}

// dockerInspect is part of docker inspect JSON
type dockerInspect struct {
	Config struct {
		Image string `json:"Image"`
	} `json:"Config"`
}

// testUpgrade test neco debian package upgrade scenario
func testUpgrade() {
	// It's only necessary for an upgrade from "without label version" to "with label version."
	// This process makes no effects even if after upgrading to "with label version."
	// However, we should delete after upgrading.
	It("should set `machine-type` label", func() {
		stdout, stderr, err := execAt(bootServers[0], "sabactl", "machines", "get")
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
		var machines []sabakan.Machine
		err = json.Unmarshal(stdout, &machines)
		Expect(err).ShouldNot(HaveOccurred(), "data=%s", stdout)
		for _, m := range machines {
			By("checking label: " + m.Spec.IPv4[0])
			if val, ok := m.Spec.Labels["machine-type"]; ok && val != "" {
				continue
			}
			By("setting label: " + m.Spec.IPv4[0])
			stdout, stderr, err := execAt(bootServers[0], "curl", "-sS", "--stderr", "-", "-X", "PUT",
				"-d", `'{ "machine-type": "qemu" }'`, "http://localhost:10080/api/v1/labels/"+m.Spec.Serial)
			Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
			Expect(string(stdout)).To(Equal(""))
		}
	})

	It("should update neco package", func() {
		data, err := os.ReadFile("../github-token")
		switch {
		case err == nil:
			By("setting github-token")

			token := string(bytes.TrimSpace(data))
			_, stderr, err := execAt(bootServers[0], "neco", "config", "set", "github-token", token)
			Expect(err).NotTo(HaveOccurred(), "stderr=%s", stderr)
			stdout, stderr, err := execAt(bootServers[0], "neco", "config", "get", "github-token")
			Expect(err).NotTo(HaveOccurred(), "stderr=%s", stderr)
			Expect(string(stdout)).To(Equal(token + "\n"))
		case os.IsNotExist(err):
		default:
			Expect(err).NotTo(HaveOccurred())
		}

		By("Changing env for test")
		stdout, stderr, err := execAt(bootServers[0], "neco", "config", "set", "env", "test")
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		By("Waiting for request to complete")
		waitRequestComplete("version: " + debVer)

		By("Checking installed Neco version")
		output := execSafeAt(bootServers[0], "dpkg-query", "--showformat=\\${Version}", "-W", neco.NecoPackageName)
		necoVersion := string(output)
		Expect(necoVersion).Should(Equal(debVer))

		By("Checking status of services enabled at postinst")
		for _, h := range bootServers {
			execSafeAt(h, "systemctl", "-q", "is-active", "neco-updater.service")
			execSafeAt(h, "systemctl", "-q", "is-active", "neco-worker.service")
			execSafeAt(h, "systemctl", "-q", "is-active", "neco-rebooter.service")
			execSafeAt(h, "systemctl", "-q", "is-active", "node-exporter.service")
			execSafeAt(h, "systemctl", "-q", "is-active", "sabakan-state-setter.service")
			execSafeAt(h, "systemctl", "-q", "is-active", "cke.service")
		}

		By("Checking version of etcd cluster")
		Eventually(func() error {
			stdout, stderr, err := execEtcdctlAt(bootServers[0], "-w", "json",
				"--endpoints=10.69.0.3:2379,10.69.0.195:2379,10.69.1.131:2379",
				"endpoint", "status")
			if err != nil {
				return fmt.Errorf("stdout=%s, stderr=%s", stdout, stderr)
			}
			var statuses []struct {
				Endpoint string `json:"Endpoint"`
				Status   struct {
					Version string `json:"version"`
				} `json:"Status"`
			}

			err = json.Unmarshal(stdout, &statuses)
			if err != nil {
				return err
			}
			for _, img := range neco.CurrentArtifacts.Images {
				if img.Name == "etcd" {
					tag := img.Tag[:strings.LastIndex(img.Tag, ".")]
					for _, s := range statuses {
						if s.Status.Version != tag {
							return errors.New("etcd is not updated: " + s.Endpoint + ", " + s.Status.Version)
						}
					}
					return nil
				}
			}
			panic("etcd image not found")
		}).Should(Succeed())
	})

	It("should re-configure vault for CKE >= 1.14.3", func() {
		stdout, stderr, err := execAt(bootServers[0], "ckecli", "--version")
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		fields := strings.Fields(string(stdout))
		Expect(fields).To(HaveLen(3))
		ver, err := version.NewVersion(fields[2])
		Expect(err).ShouldNot(HaveOccurred())

		if ver.LessThan(version.Must(version.NewVersion("1.14.3"))) {
			return
		}

		token := getVaultToken()
		stdout, stderr, err = execAt(bootServers[0], "env", "VAULT_TOKEN="+token, "ckecli", "vault", "init")
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
	})

	It("should wait for completed phase", func() {
		Eventually(func(g Gomega) {
			stdout, stderr, err := execAt(bootServers[0], "ckecli", "status")
			g.Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

			stdout, stderr, err = execAtWithInput(bootServers[0], stdout, "jq", "-r", ".phase")
			g.Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

			if strings.TrimSpace(string(stdout)) != "completed" {
				err := errors.New("CKE should complete operations")
				g.Expect(err).ShouldNot(HaveOccurred())
			}
		}).Should(Succeed())
	})

	/*It("should update cilium-agent", func() {
		stdout, stderr, err := execAt(bootServers[0], "kubectl", "delete", "pod", "-n=kube-system", "-l=app.kubernetes.io/name=cilium-agent")
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
	})*/

	It("should running newer cke desired image version", func() {
		stdout, stderr, err := execAt(bootServers[0], "ckecli", "cluster", "get")
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		cluster := new(ckeCluster)
		err = yaml.Unmarshal(stdout, cluster)
		Expect(err).ShouldNot(HaveOccurred(), "data=%s", stdout)

		By("generating kubeconfig for cluster admin")
		Eventually(func() error {
			_, stderr, err := execAt(bootServers[0], "ckecli", "kubernetes", "issue", ">", ".kube/config")
			if err != nil {
				return fmt.Errorf("err: %v, stderr: %s", err, stderr)
			}
			return nil
		}).Should(Succeed())

		stdout, stderr, err = execAt(bootServers[0], "ckecli", "images")
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		for _, img := range strings.Fields(string(stdout)) {
			By("checking " + img + " is running")
			Eventually(func() error {
				switch strings.Split(img, ":")[0] {
				case "ghcr.io/cybozu/unbound", "ghcr.io/cybozu/unbound_exporter":
					if err := checkVersionInDaemonSet("kube-system", "node-dns", img); err != nil {
						return err
					}
					if err := checkVersionInDeployment("internet-egress", "unbound", img); err != nil {
						return err
					}
					return checkVersionInDeployment("internet-egress", "squid", img)
				case "ghcr.io/cybozu/coredns":
					return checkVersionInDeployment("kube-system", "cluster-dns", img)
				case "ghcr.io/cybozu/kubernetes":
					for _, node := range cluster.Nodes {
						if node.ControlPlane {
							for _, cn := range []string{"kube-apiserver", "kube-scheduler", "kube-controller-manager"} {
								if err := checkVersionByDocker(node.Address, cn, img); err != nil {
									return err
								}
							}
						}
						if err := checkVersionByDocker(node.Address, "kubelet", img); err != nil {
							return err
						}
					}
				case "ghcr.io/cybozu/etcd":
					for _, node := range cluster.Nodes {
						if node.ControlPlane {
							if err := checkVersionByDocker(node.Address, "etcd", img); err != nil {
								return err
							}
						}
					}
				case "ghcr.io/cybozu-go/cke-tools":
					for _, node := range cluster.Nodes {
						if err := checkVersionByDocker(node.Address, "rivers", img); err != nil {
							return err
						}
					}
				case "ghcr.io/cybozu/pause":
					// Skip to check version because newer pause image is loaded after reboot
					break
				default:
					// probably this test code does not follow cke
					return errors.New("cke uses unknown container image")
				}
				return nil
			}, 19*time.Minute).Should(Succeed())
		}
	})

	It("should running newer neco desired image version", func() {
		for _, img := range neco.CurrentArtifacts.Images {
			stdout, stderr, err := execAt(bootServers[0], "neco", "image", img.Name)
			Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
			newImage := string(bytes.TrimSpace(stdout))
			By("checking " + newImage + " is running")

			Eventually(func() error {
				switch img.Name {
				case "bird", "chrony":
					// these are not managed by neco-worker
				case "cke":
					// CKE is not running as a container
				case "coil":
					if err := checkVersionInDaemonSet("kube-system", "coild", newImage); err != nil {
						return err
					}
					return checkVersionInDeployment("kube-system", "coil-controller", newImage)
				case "squid":
					return checkVersionInDeployment("internet-egress", "squid", newImage)
				case "squid-exporter":
					return checkVersionInDeployment("internet-egress", "squid", newImage)
				case "cilium":
					return nil
					//return checkVersionInDaemonSet("kube-system", "cilium", newImage)
				case "cilium-operator-generic":
					return checkVersionInDeployment("kube-system", "cilium-operator", newImage)
				case "hubble-relay":
					return checkVersionInDeployment("kube-system", "hubble-relay", newImage)
				case "cilium-certgen":
					if err := checkVersionInCronJob("kube-system", "hubble-generate-certs", newImage); err != nil {
						return err
					}
					return checkVersionInJob("kube-system", "hubble-generate-certs", newImage)
				default:
					for _, h := range bootServers {
						if _, _, err := execAt(h, "neco", "is-running", img.Name); err != nil {
							return err
						}
					}
				}
				return nil
			}).Should(Succeed())
		}
	})

	It("should SHA1 veth name is attached with newer coil", func() {
		By("deploying testhttpd")
		execSafeAt(bootServers[0], "kubectl", "run", "testhttpd", "--image=quay.io/cybozu/testhttpd:0")
		Eventually(func() error {
			stdout, _, err := execAt(bootServers[0], "kubectl", "get", "pod/testhttpd", "-o=json")
			if err != nil {
				return err
			}

			pod := new(corev1.Pod)
			err = json.Unmarshal(stdout, pod)
			if err != nil {
				return err
			}

			if pod.Status.Phase != corev1.PodRunning {
				return fmt.Errorf("Pod is not running: %s", pod.Status.Phase)
			}
			return nil
		}).Should(Succeed())

		stdout, stderr, err := execAt(bootServers[0], "kubectl", "get", "pod/testhttpd", "-o=json")
		Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
		pod := new(corev1.Pod)
		err = json.Unmarshal(stdout, pod)
		Expect(err).NotTo(HaveOccurred(), "data=%s", stdout)
		By("checking SHA1 veth for namespace: " + pod.Namespace + ", name:" + pod.Name)
		checkVethPeerNameIsSHA1(pod)

		execSafeAt(bootServers[0], "kubectl", "delete", "pod/testhttpd")
	})

	It("should SHA1 veth name is attached when container restarts with newer coil", func() {
		By("stopping a squid pod")
		const squidNum = 3
		var podName, notTarget string
		Eventually(func() error {
			stdout, stderr, err := execAt(bootServers[0], "kubectl", "-n=internet-egress", "get", "pods", "--selector=app.kubernetes.io/name=squid", "-o=json")
			if err != nil {
				return fmt.Errorf("err: %v, stdout:%s, stderr: %s", err, stdout, stderr)
			}
			podList := new(corev1.PodList)
			err = json.Unmarshal(stdout, podList)
			if err != nil {
				return err
			}
			if len(podList.Items) < squidNum {
				return fmt.Errorf("len(podList.Items) < 2, actual: %d", len(podList.Items))
			}
			podName = podList.Items[0].Name
			notTarget = podList.Items[1].Name
			return nil
		}).Should(Succeed())

		// Eventually() will be no longer needed by https://github.com/cybozu-go/neco/issues/429
		Eventually(func() error {
			stdout, stderr, err := execAt(bootServers[0], "kubectl", "-n=internet-egress", "delete", "pod", podName)
			if err != nil {
				return fmt.Errorf("err: %v, stdout:%s, stderr: %s", err, stdout, stderr)
			}
			return nil
		}).Should(Succeed())

		By("waiting squid deployment is ready")
		Eventually(func() error {
			stdout, stderr, err := execAt(bootServers[0], "kubectl", "-n=internet-egress", "get", "deployment", "squid", "-o=json")
			if err != nil {
				return fmt.Errorf("%v: stderr=%s", err, stderr)
			}

			deployment := new(appsv1.Deployment)
			err = json.Unmarshal(stdout, deployment)
			if err != nil {
				return err
			}

			if int(deployment.Status.AvailableReplicas) != squidNum {
				return fmt.Errorf("AvailableReplicas is not %d but %d", squidNum, deployment.Status.AvailableReplicas)
			}
			return nil
		}).Should(Succeed())

		stdout, stderr, err := execAt(bootServers[0], "kubectl", "-n=internet-egress", "get", "pods", "--selector=app.kubernetes.io/name=squid", "-o=json")
		Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
		podList := new(corev1.PodList)
		err = json.Unmarshal(stdout, podList)
		Expect(err).NotTo(HaveOccurred(), "data=%s", stdout)

		for _, pod := range podList.Items {
			if pod.Name != notTarget {
				By("checking SHA1 veth for namespace: " + pod.Namespace + ", name:" + pod.Name)
				checkVethPeerNameIsSHA1(&pod)
				break
			}
		}
	})
}

func checkVersionByDocker(address, name, image string) error {
	stdout, stderr, err := execAt(bootServers[0], "ckecli", "ssh", address, "docker", "inspect", name)
	if err != nil {
		return fmt.Errorf("stderr: %s, err: %v", stderr, err)
	}

	var dis []dockerInspect
	err = json.Unmarshal(stdout, &dis)
	if err != nil {
		return err
	}

	for _, di := range dis {
		if image != di.Config.Image {
			return fmt.Errorf("desired image: %s, actual image: %s", image, di.Config.Image)
		}
	}
	return nil
}

func checkVersionInDaemonSet(namespace, dsName, image string) error {
	stdout, _, err := execAt(bootServers[0], "kubectl", "get", "ds", "-n", namespace, dsName, "-o", "json")
	if err != nil {
		return err
	}
	ds := new(appsv1.DaemonSet)
	err = json.Unmarshal(stdout, ds)
	if err != nil {
		return err
	}
	found := false
	for _, c := range ds.Spec.Template.Spec.Containers {
		if c.Image == image {
			found = true
		}
	}
	if !found {
		return fmt.Errorf("%s not found in %s", image, dsName)
	}
	if ds.Status.DesiredNumberScheduled != ds.Status.NumberAvailable {
		return fmt.Errorf("%s %s is not updated completely. desired number scheduled is %d, but actual available is %d",
			dsName, image, ds.Status.DesiredNumberScheduled, ds.Status.NumberAvailable)
	}
	if ds.Status.DesiredNumberScheduled != ds.Status.UpdatedNumberScheduled {
		return fmt.Errorf("%s %s is not updated completely. desired number scheduled is %d, but actual updated is %d",
			dsName, image, ds.Status.DesiredNumberScheduled, ds.Status.UpdatedNumberScheduled)
	}
	return nil
}

func checkVersionInDeployment(namespace, deploymentName, image string) error {
	stdout, _, err := execAt(bootServers[0], "kubectl", "get", "deployment", "-n", namespace, deploymentName, "-o", "json")
	if err != nil {
		return err
	}
	deploy := new(appsv1.Deployment)
	err = json.Unmarshal(stdout, deploy)
	if err != nil {
		return err
	}
	found := false
	for _, c := range deploy.Spec.Template.Spec.Containers {
		if c.Image == image {
			found = true
		}
	}
	if !found {
		return fmt.Errorf("%s not found in %s", image, deploymentName)
	}
	desired := int32(1)
	if deploy.Spec.Replicas != nil {
		desired = *deploy.Spec.Replicas
	}
	if actual := deploy.Status.AvailableReplicas; actual != desired {
		return fmt.Errorf("%s's %s is not updated completely. desired replicas is %d, but actual available is %d",
			deploymentName, image, desired, actual)
	}
	if actual := deploy.Status.UpdatedReplicas; actual != desired {
		return fmt.Errorf("%s's %s is not updated completely. desired replicas is %d, but actual updated is %d",
			deploymentName, image, desired, actual)
	}
	return nil
}

func checkVersionInCronJob(namespace, cjName, image string) error {
	stdout, _, err := execAt(bootServers[0], "kubectl", "get", "cj", "-n", namespace, cjName, "-o", "json")
	if err != nil {
		return err
	}

	cj := new(batchv1beta1.CronJob)
	err = json.Unmarshal(stdout, cj)
	if err != nil {
		return err
	}
	found := false
	for _, c := range cj.Spec.JobTemplate.Spec.Template.Spec.Containers {
		if c.Image == image {
			found = true
		}
	}
	if !found {
		return fmt.Errorf("%s not found in %s", image, cjName)
	}
	return nil
}

func checkVersionInJob(namespace, jobPrefix, image string) error {
	stdout, _, err := execAt(bootServers[0], "kubectl", "get", "job", "-n", namespace, "-o", "json")
	if err != nil {
		return err
	}

	jobList := new(batchv1.JobList)
	if err := json.Unmarshal(stdout, jobList); err != nil {
		return err
	}

	found := false
	var jobs []batchv1.Job
	for _, j := range jobList.Items {
		if strings.HasPrefix(j.Name, jobPrefix) {
			found = true
			jobs = append(jobs, j)
		}
	}
	if !found {
		return fmt.Errorf("%s not found", jobPrefix)
	}

	found = false
	for _, job := range jobs {
		for _, c := range job.Spec.Template.Spec.Containers {
			if c.Image == image {
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		return fmt.Errorf("%s not found in %v", image, jobs)
	}
	return nil
}

func checkVethPeerNameIsSHA1(pod *corev1.Pod) {
	h := sha1.New()
	h.Write([]byte(fmt.Sprintf("%s.%s", pod.Namespace, pod.Name)))
	peerName := fmt.Sprintf("%s%s", "veth", hex.EncodeToString(h.Sum(nil))[:11])
	execSafeAt(bootServers[0], "ckecli", "ssh", pod.Status.HostIP, "ip", "link", "show", peerName)
}
