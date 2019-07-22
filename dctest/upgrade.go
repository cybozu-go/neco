package dctest

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/sabakan/v2"
	"github.com/hashicorp/go-version"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	yaml "gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// ckeCluster is part of cke.Cluster in github.com/cybozu-go/cke
type ckeCluster struct {
	Nodes []*ckeNode `yaml:"nodes"`
}

// ckeNode is part of cke.Node in github.com/cybozu-go/cke
type ckeNode struct {
	Address      string `yaml:"address"`
	ControlPlane bool   `yaml:"control_plane"`
}

// dockerInspect is part of docker inspect JSON
type dockerInspect struct {
	Config struct {
		Image string `json:"Image"`
	} `json:"Config"`
}

// rktManifest is part of rkt cat-manifest JSON
type rktManifest struct {
	Apps []struct {
		Name  string `json:"name"`
		Image struct {
			Name   string `json:"name"`
			Labels []struct {
				Name  string `json:"name"`
				Value string `json:"value"`
			} `json:"labels"`
		} `json:"image"`
	} `json:"apps"`
}

// TestUpgrade test neco debian package upgrade scenario
func TestUpgrade() {
	// It's only necessary for an upgrade from "without label version" to "with label version."
	// This process makes no effects even if after upgrading to "with label version."
	// However, we should delete after upgrading.
	It("should set `machine-type` label", func() {
		stdout, stderr, err := execAt(boot0, "sabactl", "machines", "get")
		Expect(err).ShouldNot(HaveOccurred(), "stderr=%s", stderr)
		var machines []sabakan.Machine
		err = json.Unmarshal(stdout, &machines)
		Expect(err).ShouldNot(HaveOccurred())
		for _, m := range machines {
			By("checking label: " + m.Spec.IPv4[0])
			if val, ok := m.Spec.Labels["machine-type"]; ok && val != "" {
				continue
			}
			By("setting label: " + m.Spec.IPv4[0])
			stdout, _, err := execAt(boot0, "curl", "-sS", "--stderr", "-", "-X", "PUT",
				"-d", `'{ "machine-type": "qemu" }'`, "http://localhost:10080/api/v1/labels/"+m.Spec.Serial)
			Expect(err).ShouldNot(HaveOccurred(), "stderr=%s", stderr)
			Expect(string(stdout)).To(Equal(""))
		}
	})

	It("should update neco package", func() {
		data, err := ioutil.ReadFile("../github-token")
		switch {
		case err == nil:
			By("setting github-token")

			token := string(bytes.TrimSpace(data))
			_, _, err = execAt(boot0, "neco", "config", "set", "github-token", token)
			Expect(err).NotTo(HaveOccurred())
			stdout, _, err := execAt(boot0, "neco", "config", "get", "github-token")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(stdout)).To(Equal(token + "\n"))
		case os.IsNotExist(err):
		default:
			Expect(err).NotTo(HaveOccurred())
		}

		By("Changing env for test")
		stdout, stderr, err := execAt(boot0, "neco", "config", "set", "env", "test")
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		By("Waiting for request to complete")
		waitRequestComplete("version: " + debVer)

		By("Checking installed Neco version")
		output := execSafeAt(boot0, "dpkg-query", "--showformat=\\${Version}", "-W", neco.NecoPackageName)
		necoVersion := string(output)
		Expect(necoVersion).Should(Equal(debVer))

		By("Checking status of services enabled at postinst")
		for _, h := range []string{boot0, boot1, boot2} {
			execSafeAt(h, "systemctl", "-q", "is-active", "neco-updater.service")
			execSafeAt(h, "systemctl", "-q", "is-active", "neco-worker.service")
			execSafeAt(h, "systemctl", "-q", "is-active", "node-exporter.service")
		}

		By("Checking version of CKE")
		Eventually(func() error {
			ckeVersion, _, err := execAt(boot0, "ckecli", "--version")
			if err != nil {
				return err
			}
			for _, img := range neco.CurrentArtifacts.Images {
				if img.Name == "cke" {
					if !bytes.Contains(ckeVersion, []byte(img.Tag)) {
						return errors.New("cke is not updated: " + string(ckeVersion))
					}
					return nil
				}
			}
			panic("cke image not found")
		}).Should(Succeed())

		By("Checking version of etcd cluster")
		Eventually(func() error {
			stdout, stderr, err := execAt(boot0, "env", "ETCDCTL_API=3", "etcdctl", "-w", "json",
				"--cert=/etc/neco/etcd.crt", "--key=/etc/neco/etcd.key",
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
		stdout, _, err := execAt(boot0, "ckecli", "--version")
		Expect(err).ShouldNot(HaveOccurred())

		fields := strings.Fields(string(stdout))
		Expect(fields).To(HaveLen(3))
		ver, err := version.NewVersion(fields[2])
		Expect(err).ShouldNot(HaveOccurred())

		if ver.LessThan(version.Must(version.NewVersion("1.14.3"))) {
			return
		}

		token := getVaultToken()
		_, _, err = execAt(boot0, "env", "VAULT_TOKEN="+token, "ckecli", "vault", "init")
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("should running newer cke desired image version", func() {
		stdout, stderr, err := execAt(boot0, "ckecli", "cluster", "get")
		Expect(err).ShouldNot(HaveOccurred(), "stderr=%s", stderr)

		cluster := new(ckeCluster)
		err = yaml.Unmarshal(stdout, cluster)
		Expect(err).ShouldNot(HaveOccurred())

		By("generating kubeconfig for cluster admin")
		execSafeAt(boot0, "mkdir", "-p", ".kube")
		execSafeAt(boot0, "ckecli", "kubernetes", "issue", ">", ".kube/config")

		stdout, stderr, err = execAt(boot0, "ckecli", "images")
		Expect(err).ShouldNot(HaveOccurred(), "stderr=%s", stderr)

		for _, img := range strings.Fields(string(stdout)) {
			By("checking " + img + " is running")
			Eventually(func() error {
				switch strings.Split(img, ":")[0] {
				case "quay.io/cybozu/unbound":
					if err := checkVersionInDaemonSet("kube-system", "node-dns", img); err != nil {
						return err
					}
					return checkVersionInDeployment("internet-egress", "unbound", img)
				case "quay.io/cybozu/coredns":
					return checkVersionInDeployment("kube-system", "cluster-dns", img)
				case "quay.io/cybozu/hyperkube":
					for _, node := range cluster.Nodes {
						if node.ControlPlane {
							for _, cn := range []string{"kube-apiserver", "kube-scheduler", "kube-controller-manager"} {
								if err := checkVersionByDocker(node.Address, cn, img); err != nil {
									return err
								}
							}
						}
						for _, cn := range []string{"kube-proxy", "kubelet"} {
							if err := checkVersionByDocker(node.Address, cn, img); err != nil {
								return err
							}
						}
					}
				case "quay.io/cybozu/etcd":
					for _, node := range cluster.Nodes {
						if node.ControlPlane {
							if err := checkVersionByDocker(node.Address, "etcd", img); err != nil {
								return err
							}
						}
					}
				case "quay.io/cybozu/cke-tools":
					for _, node := range cluster.Nodes {
						if err := checkVersionByDocker(node.Address, "rivers", img); err != nil {
							return err
						}
					}
				case "quay.io/cybozu/pause":
					// Skip to check version because newer pause image is loaded after reboot
					break
				}
				return nil
			}, 20*time.Minute).Should(Succeed())
		}
	})

	It("should running newer neco desired image version", func() {
		for _, img := range neco.CurrentArtifacts.Images {
			stdout, stderr, err := execAt(boot0, "neco", "image", img.Name)
			Expect(err).ShouldNot(HaveOccurred(), "stderr=%s", stderr)
			newImage := string(bytes.TrimSpace(stdout))
			By("checking " + newImage + " is running")

			Eventually(func() error {
				switch img.Name {
				case "coil":
					if err := checkVersionInDaemonSet("kube-system", "coil-node", newImage); err != nil {
						return err
					}
					return checkVersionInDeployment("kube-system", "coil-controllers", newImage)
				case "squid":
					return checkVersionInDeployment("internet-egress", "squid", newImage)
				case "teleport":
					for _, h := range []string{boot0, boot1, boot2} {
						if err := checkVersionOfTeleport(h, newImage); err != nil {
							return err
						}
					}
				default:
					for _, h := range []string{boot0, boot1, boot2} {
						if err := checkVersionByRkt(h, newImage); err != nil {
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
		execSafeAt(boot0, "kubectl", "run", "testhttpd", "--image=quay.io/cybozu/testhttpd:0")
		Eventually(func() error {
			stdout, _, err := execAt(boot0, "kubectl", "get", "deployments/testhttpd", "-o=json")
			if err != nil {
				return err
			}

			deployment := new(appsv1.Deployment)
			err = json.Unmarshal(stdout, deployment)
			if err != nil {
				return err
			}

			if int(deployment.Status.AvailableReplicas) != 1 {
				return fmt.Errorf("AvailableReplicas is not 1: %d", int(deployment.Status.AvailableReplicas))
			}
			return nil
		}).Should(Succeed())

		stdout, stderr, err := execAt(boot0, "kubectl", "get", "pods", "--selector=run=testhttpd", "-o=json")
		Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)
		podList := new(corev1.PodList)
		err = json.Unmarshal(stdout, podList)
		Expect(err).NotTo(HaveOccurred())
		for _, pod := range podList.Items {
			By("checking SHA1 veth for namespace: " + pod.Namespace + ", name:" + pod.Name)
			checkVethPeerNameIsSHA1(&pod)
		}
		execSafeAt(boot0, "kubectl", "delete", "deployments/testhttpd")
	})

	It("should SHA1 veth name is attached when container restarts with newer coil", func() {
		By("stopping a squid pod")
		stdout, stderr, err := execAt(boot0, "kubectl", "-n=internet-egress", "get", "pods", "--selector=k8s-app=squid", "-o=json")
		Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)
		podList := new(corev1.PodList)
		err = json.Unmarshal(stdout, podList)
		Expect(err).NotTo(HaveOccurred())
		podName := podList.Items[0].Name
		notTarget := podList.Items[1].Name
		_, stderr, err = execAt(boot0, "kubectl", "-n=internet-egress", "delete", "pod", podName)
		Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)

		By("waiting squid deployment is ready")
		Eventually(func() error {
			stdout, stderr, err := execAt(boot0, "kubectl", "-n=internet-egress", "get", "deployment", "squid", "-o=json")
			if err != nil {
				return fmt.Errorf("%v: stderr=%s", err, stderr)
			}

			deployment := new(appsv1.Deployment)
			err = json.Unmarshal(stdout, deployment)
			if err != nil {
				return err
			}

			if int(deployment.Status.AvailableReplicas) != 2 {
				return errors.New("AvailableReplicas is not 2")
			}
			return nil
		}).Should(Succeed())

		stdout, stderr, err = execAt(boot0, "kubectl", "-n=internet-egress", "get", "pods", "--selector=k8s-app=squid", "-o=json")
		Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)
		podList = new(corev1.PodList)
		err = json.Unmarshal(stdout, podList)
		Expect(err).NotTo(HaveOccurred())

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
	stdout, stderr, err := execAt(boot0, "ckecli", "ssh", address, "docker", "inspect", name)
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

func checkVersionOfTeleport(host, image string) error {
	// this returns a string like "Teleport v4.0.2 git:v4.0.2-0-gb7e0e872 go1.12.5"
	stdout, stderr, err := execAt(host, "teleport", "version")
	if err != nil {
		return fmt.Errorf("host: %s, stderr: %s, err: %v", host, stderr, err)
	}

	var version, s1, s2 string
	n, err := fmt.Sscanf(string(stdout), "Teleport v%s %s %s", &version, &s1, &s2)
	if err != nil || n != 3 {
		return fmt.Errorf("unexpected version format: %s", stdout)
	}

	expected := strings.Split(image, ":")[1]
	if !strings.HasPrefix(expected, version+".") {
		return fmt.Errorf("unexpected version: %s", version)
	}

	return nil
}

func checkVersionByRkt(host, image string) error {
	fullName := strings.Split(image, ":")[0]
	shortName := strings.TrimPrefix(fullName, "quay.io/cybozu/")
	version := strings.Split(image, ":")[1]

	stdout, stderr, err := execAt(host, "sudo", "rkt", "list", "--format", "json")
	if err != nil {
		return fmt.Errorf("host: %s, stderr: %s, err: %v", host, stderr, err)
	}

	var pods []neco.RktPod
	err = json.Unmarshal(stdout, &pods)
	if err != nil {
		return err
	}

	if len(pods) == 0 {
		return fmt.Errorf("failed to get pod list at %s", host)
	}

	for _, pod := range pods {
		if pod.State != "running" {
			continue
		}

		var uuid string
		for _, appName := range pod.AppNames {
			if appName == shortName {
				uuid = pod.Name
			}
		}

		if len(uuid) == 0 {
			continue
		}

		stdout, stderr, err := execAt(host, "sudo", "rkt", "cat-manifest", uuid)
		if err != nil {
			return fmt.Errorf("host: %s, stderr: %s, err: %v", host, stderr, err)
		}

		var manifest rktManifest
		err = json.Unmarshal(stdout, &manifest)
		if err != nil {
			return err
		}

		found := false
		for _, app := range manifest.Apps {
			if app.Image.Name == fullName {
				for _, label := range app.Image.Labels {
					if label.Name == "version" && label.Value == version {
						found = true
					}
				}
			}
		}

		if found {
			return nil
		}
	}

	return fmt.Errorf("desired image %s is not running at %s", image, host)
}

func checkVersionInDaemonSet(namespace, dsName, image string) error {
	stdout, _, err := execAt(boot0, "kubectl", "get", "ds", "-n", namespace, dsName, "-o", "json")
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
	stdout, _, err := execAt(boot0, "kubectl", "get", "deployment", "-n", namespace, deploymentName, "-o", "json")
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

func checkVethPeerNameIsSHA1(pod *corev1.Pod) {
	h := sha1.New()
	h.Write([]byte(fmt.Sprintf("%s.%s", pod.Namespace, pod.Name)))
	peerName := fmt.Sprintf("%s%s", "veth", hex.EncodeToString(h.Sum(nil))[:11])
	execSafeAt(boot0, "ckecli", "ssh", pod.Status.HostIP, "ip", "link", "show", peerName)
}
