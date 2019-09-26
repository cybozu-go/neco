package dctest

import (
	"encoding/json"
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// TestSerfTagsProxy tests modifying serf tags.
func TestSerfTagsProxy() {
	dummyProxyURL := "http://0.0.0.0:30128"

	dockerPIDs := make(map[string]string)
	containerdPIDs := make(map[string]string)

	It("should get current PIDs of container runtime services", func() {
		members, err := getSerfMembers()
		Expect(err).NotTo(HaveOccurred())
		for _, m := range members.Members {
			addr := strings.Split(m.Addr, ":")[0]
			stdout := execSafeAt(boot0, "ckecli", "ssh", addr, "--", "systemctl", "show", "-p", "ExecMainPID", "--value", "docker.service")
			dockerPIDs[addr] = strings.TrimSpace(string(stdout))
			stdout = execSafeAt(boot0, "ckecli", "ssh", addr, "--", "systemctl", "show", "-p", "ExecMainPID", "--value", "k8s-containerd.service")
			containerdPIDs[addr] = strings.TrimSpace(string(stdout))
		}
	})

	It("should set workers' proxy configuration to serf tag", func() {
		execSafeAt(boot0, "neco", "serf-tag", "set", "proxy", dummyProxyURL)
		Eventually(func() error {
			stdout, stderr, err := execAt(boot0, "serf", "members", "-format", "json", "-tag", "boot-server=true")
			if err != nil {
				return fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
			}
			var m serfMemberContainer
			err = json.Unmarshal(stdout, &m)
			if err != nil {
				return err
			}
			if len(m.Members) != 3 {
				return fmt.Errorf("serf members should be 3: %d", len(m.Members))
			}

			for _, member := range m.Members {
				tag, ok := member.Tags["proxy"]
				if !ok {
					return fmt.Errorf("member %s does not define tag proxy", member.Name)
				}
				if tag != dummyProxyURL {
					return fmt.Errorf("member %s should have proxy %s: %s", member.Name, dummyProxyURL, tag)
				}
			}
			return nil
		}).Should(Succeed())
	})

	It("should confirm change of workers' proxy configuration", func() {
		Eventually(func() error {
			for addr := range dockerPIDs {
				stdout := execSafeAt(boot0, "ckecli", "ssh", addr, "cat", "/etc/neco/proxy.env")
				if string(stdout) != fmt.Sprintf("HTTP_PROXY=%s\nHTTPS_PROXY=%s\n", dummyProxyURL, dummyProxyURL) {
					return fmt.Errorf("member %s have not reflected tag proxy", addr)
				}
			}
			return nil
		}).Should(Succeed())
	})

	It("should confirm change of PIDs of container runtime services", func() {
		Eventually(func() error {
			for k, v := range dockerPIDs {
				stdout := execSafeAt(boot0, "ckecli", "ssh", k, "--", "systemctl", "show", "-p", "ExecMainPID", "--value", "docker.service")
				if strings.TrimSpace(string(stdout)) == v {
					return fmt.Errorf("member %s have not restarted docker.service", k)
				}
			}

			for k, v := range containerdPIDs {
				stdout = execSafeAt(boot0, "ckecli", "ssh", k, "--", "systemctl", "show", "-p", "ExecMainPID", "--value", "k8s-containerd.service")
				if strings.TrimSpace(string(stdout)) == v {
					return fmt.Errorf("member %s have not restarted k8s-containerd.service", k)
				}
			}
			return nil
		}).Should(Succeed())
	})

	It("should confirm change of configuration of container runtime services", func() {
		for addr := range dockerPIDs {
			stdout := execSafeAt(boot0, "ckecli", "ssh", addr, "--", "docker", "-D", "info", "--format", "{{.HTTPProxy}}")
			Expect(strings.TrimSpace(string(stdout))).To(Equal(dummyProxyURL))
			stdout = execSafeAt(boot0, "ckecli", "ssh", addr, "--", "docker", "-D", "info", "--format", "{{.HTTPSProxy}}")
			Expect(strings.TrimSpace(string(stdout))).To(Equal(dummyProxyURL))
		}
		// skip test for containerd because we cannot find CLI to get configuration
	})
}
