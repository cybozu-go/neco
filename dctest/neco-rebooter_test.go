package dctest

import (
	"errors"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func testNecoRebooter() {
	It("should switch the leader", func() {
		By("getting leader node name")
		var leaderNodeBefore string
		Eventually(func() error {
			stdout, _, err := execAt(bootServers[0], "neco", "rebooter", "leader")
			if err != nil {
				return err
			}
			if len(stdout) == 0 {
				return errors.New("no leader")
			}
			leaderNodeBefore = strings.TrimSuffix(string(stdout), "\n")
			return nil
		}).Should(Succeed())

		By("restarting neco-rebooter on " + leaderNodeBefore)
		index, err := strconv.Atoi(leaderNodeBefore[len(leaderNodeBefore)-1:])
		Expect(err).ShouldNot(HaveOccurred(), "data=%s", leaderNodeBefore[len(leaderNodeBefore)-1:])
		stdout, stderr, err := execAt(bootServers[index], "sudo", "systemctl", "restart", "neco-rebooter.service")
		Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		By("getting leader node name again")
		Eventually(func() error {
			stdout, _, err := execAt(bootServers[0], "neco", "rebooter", "leader")
			if err != nil {
				return err
			}
			if len(stdout) == 0 {
				return errors.New("no leader")
			}
			leaderNodeAfter := strings.TrimSuffix(string(stdout), "\n")
			if leaderNodeAfter == leaderNodeBefore {
				return errors.New("leader is not changed")
			}
			return nil
		}).Should(Succeed())
	})
}
