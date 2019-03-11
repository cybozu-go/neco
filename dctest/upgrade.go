package dctest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

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
	It("should install desired version", func() {
		stdout, stderr, err := execAt(boot0, "ckecli", "images")
		Expect(err).ShouldNot(HaveOccurred(), "stderr=%s", stderr)

		for _, img := range strings.Split(string(stdout), "\n") {
			split := strings.Split(img[len("quay.io/cybozu/"):], ":")
			imageName := split[0]
			version := split[1]

			// TODO: check all images
			if imageName != "unbound" && imageName != "coredns" {
				continue
			}
			err := checkRunningDesiredVersion(imageName, version)
			Expect(err).ShouldNot(HaveOccurred())
		}
	})
}

func checkRunningDesiredVersion(imageName string, desiredVersion string) error {
	switch imageName {
	case "unbound":
		// node-dns
		err := checkVersionInDaemonSet("kube-system", "node-dns", imageName, desiredVersion)
		if err != nil {
			return err
		}
		// internet full resolver
		err = checkVersionInDeployment("internet-egress", "unbound", imageName, desiredVersion)
		if err != nil {
			return err
		}
		// unbound in squid pod
		err = checkVersionInDeployment("internet-egress", "squid", imageName, desiredVersion)
		if err != nil {
			return err
		}
	case "coredns":
		err := checkVersionInDeployment("kube-system", "cluster-dns", imageName, desiredVersion)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("checker function for %s is not implemented", imageName)
	}
	return nil
}

func checkVersionInDaemonSet(namespace, dsName, imageName, desiredVersion string) error {
	stdout, _, err := execAt(boot0, "kubectl", "get", "ds", "-n", namespace, dsName, "-o", "json")
	if err != nil {
		return err
	}
	ds := new(appsv1.DaemonSet)
	err = json.Unmarshal(stdout, ds)
	if err != nil {
		return err
	}
	actual := ds.GetObjectMeta().GetAnnotations()["cke.cybozu.com/image"][len("quay.io/cybozu/:"+imageName):]
	if actual != desiredVersion {
		return fmt.Errorf("%s %s is not updated. desired version is %s, but actual version is %s",
			dsName, imageName, desiredVersion, actual)
	}
	if ds.Status.DesiredNumberScheduled != ds.Status.NumberAvailable {
		return fmt.Errorf("%s %s is not updated completely. desired number scheduled is %d, but actual available is %d",
			dsName, imageName, ds.Status.DesiredNumberScheduled, ds.Status.NumberAvailable)
	}
	return nil
}

func checkVersionInDeployment(namespace, deploymentName, imageName, desiredVersion string) error {
	stdout, _, err := execAt(boot0, "kubectl", "get", "deployment", "-n", namespace, deploymentName, "-o", "json")
	if err != nil {
		return err
	}
	deploy := new(appsv1.Deployment)
	err = json.Unmarshal(stdout, deploy)
	if err != nil {
		return err
	}
	for _, c := range deploy.Spec.Template.Spec.Containers {
		if !strings.HasPrefix(c.Image, "quay.io/cybozu/"+imageName) {
			continue
		}
		if actual := c.Image[len("quay.io/cybozu/"+imageName):]; actual != desiredVersion {
			return fmt.Errorf("%s's %s is not updated. desired version is %s, but actual version is %s",
				deploymentName, imageName, desiredVersion, actual)
		}
	}
	if actual := deploy.Status.AvailableReplicas; actual != *deploy.Spec.Replicas {
		return fmt.Errorf("%s's %s is not updated completely. desired replicas is %d, but actual available is %d",
			deploymentName, imageName, deploy.Spec.Replicas, actual)
	}
	return nil
}
