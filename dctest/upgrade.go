package dctest

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/cybozu-go/neco"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	yaml "gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
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
		_, _, err = execAt(boot0, "neco", "config", "set", "env", "test")
		Expect(err).ShouldNot(HaveOccurred())

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
	})

	It("should generate encryption key for CKE 1.13.17", func() {
		stdout, _, err := execAt(boot0, "ckecli", "--version")
		Expect(err).ShouldNot(HaveOccurred())

		if !bytes.Contains(stdout, []byte("1.13.17")) {
			return
		}

		_, _, err = execAt(boot0, "ckecli", "vault", "enckey")
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("should running newer cke desired image version", func() {
		stdout, stderr, err := execAt(boot0, "ckecli", "cluster", "get")
		Expect(err).ShouldNot(HaveOccurred(), "stderr=%s", stderr)

		cluster := new(ckeCluster)
		err = yaml.Unmarshal(stdout, cluster)
		Expect(err).ShouldNot(HaveOccurred())

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
			}).Should(Succeed())
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
					if err := checkVersionInDaemonSet("kube-system", "coil-node", string(stdout)); err != nil {
						return err
					}
					return checkVersionInDeployment("kube-system", "coil-controller", string(stdout))
				case "squid":
					return checkVersionInDeployment("internet-egress", "squid", string(stdout))
				default:
					for _, h := range []string{boot0, boot1, boot2, boot3} {
						if err := checkVersionByRkt(h, string(stdout)); err != nil {
							return err
						}
					}
				}
				return nil
			}).Should(Succeed())
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

func checkVersionByRkt(address, image string) error {
	fullName := strings.Split(image, ":")[0]
	fmt.Println(fullName)
	shortName := strings.TrimPrefix(fullName, "quay.io/cybozu/")
	fmt.Println(shortName)
	version := strings.Split(image, ":")[1]
	fmt.Println(version)

	stdout, stderr, err := execAt(address, "sudo", "rkt", "list", "--format", "json")
	if err != nil {
		return fmt.Errorf("stderr: %s, err: %v", stderr, err)
	}

	var pods []neco.RktPod
	err = json.Unmarshal(stdout, &pods)
	if err != nil {
		return err
	}

	if len(pods) == 0 {
		return errors.New("failed to get pod list")
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

		stdout, stderr, err := execAt(address, "sudo", "rkt", "cat-manifest", uuid)
		if err != nil {
			return fmt.Errorf("stderr: %s, err: %v", stderr, err)
		}

		var manifest rktManifest
		err = json.Unmarshal(stdout, &manifest)
		if err != nil {
			return err
		}

		nameFound := false
		versionFound := false
		for _, app := range manifest.Apps {
			if app.Image.Name == fullName {
				nameFound = true
			}

			for _, label := range app.Image.Labels {
				if label.Name == "version" && label.Value == version {
					versionFound = true
				}
			}
		}

		if !nameFound || !versionFound {
			return fmt.Errorf("desired image is not running: %v", image)
		}
	}

	return nil
}

func checkRunningDesiredVersion(image string) error {
	if strings.Contains(image, "unbound") {
		// node-dns
		err := checkVersionInDaemonSet("kube-system", "node-dns", image)
		if err != nil {
			return err
		}
		// internet full resolver
		err = checkVersionInDeployment("internet-egress", "unbound", image)
		if err != nil {
			return err
		}
		// unbound in squid pod
		err = checkVersionInDeployment("internet-egress", "squid", image)
		if err != nil {
			return err
		}
	} else if strings.Contains(image, "coredns") {
		err := checkVersionInDeployment("kube-system", "cluster-dns", image)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("checking %s is not implemented", image)
	}
	return nil
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
