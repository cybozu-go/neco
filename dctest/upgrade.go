package dctest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/cybozu-go/neco"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
)

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

	It("should install desired version", func() {
		stdout, stderr, err := execAt(boot0, "ckecli", "images")
		Expect(err).ShouldNot(HaveOccurred(), "stderr=%s", stderr)

		for _, img := range strings.Fields(string(stdout)) {
			// TODO: check all images
			if !strings.Contains(img, "unbound") && !strings.Contains(img, "coredns") {
				continue
			}
			Eventually(func() error {
				err := checkRunningDesiredVersion(img)
				return err
			}, 20*time.Minute).Should(Succeed())
		}
	})
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
