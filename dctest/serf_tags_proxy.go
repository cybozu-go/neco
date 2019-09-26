package dctest

import (
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestSerfTagsProxy() {
	dummyProxyURL := "http://0.0.0.0:30128"

	It("should set surf-tags-proxy", func() {
		execSafeAt(boot0, "neco", "config", "set", "serf-tags-proxy", dummyProxyURL)
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
}
