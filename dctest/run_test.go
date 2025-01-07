package dctest

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	. "github.com/onsi/gomega"
	"golang.org/x/crypto/ssh"
)

const (
	sshTimeout = 5 * time.Minute

	defaultDialTimeout = 30 * time.Second
	defaultKeepAlive   = 5 * time.Second

	// DefaultRunTimeout is the timeout value for Agent.Run().
	DefaultRunTimeout = 10 * time.Minute

	proxy                    = "http://10.0.49.3:3128"
	externalIPBlock          = "172.19.0.16/28"
	lbAddressBlockDefault    = "10.72.32.0/20"
	lbAddressBlockBastion    = "10.72.48.48/28"
	lbAddressBlockInternet   = "172.19.0.16/28"
	lbAddressBlockInternetCN = "172.19.0.248/29"
)

var (
	sshClients = make(map[string]*sshAgent)

	// https://github.com/cybozu-go/cke/pull/81/files
	agentDialer = &net.Dialer{
		Timeout:   defaultDialTimeout,
		KeepAlive: defaultKeepAlive,
	}
)

type retryHandler func(stdout, stderr string, err error) bool

type sshAgent struct {
	client *ssh.Client
	conn   net.Conn
}

func sshTo(address string, sshKey ssh.Signer, userName string) (*sshAgent, error) {
	conn, err := agentDialer.Dial("tcp", address+":22")
	if err != nil {
		fmt.Printf("failed to dial: %s\n", address)
		return nil, err
	}
	config := &ssh.ClientConfig{
		User: userName,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(sshKey),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}
	err = conn.SetDeadline(time.Now().Add(defaultDialTimeout))
	if err != nil {
		conn.Close()
		return nil, err
	}
	clientConn, channelCh, reqCh, err := ssh.NewClientConn(conn, "tcp", config)
	if err != nil {
		// conn was already closed in ssh.NewClientConn
		return nil, err
	}
	err = conn.SetDeadline(time.Time{})
	if err != nil {
		clientConn.Close()
		return nil, err
	}
	a := sshAgent{
		client: ssh.NewClient(clientConn, channelCh, reqCh),
		conn:   conn,
	}
	return &a, nil
}

func parsePrivateKey(keyPath string) (ssh.Signer, error) {
	f, err := os.Open(keyPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	return ssh.ParsePrivateKey(data)
}

func prepareSSHClients(addresses ...string) error {
	sshKey, err := parsePrivateKey(sshKeyFile)
	if err != nil {
		return err
	}

	ch := time.After(sshTimeout)
	for _, a := range addresses {
	RETRY:
		select {
		case <-ch:
			return errors.New("prepareSSHClients timed out")
		default:
		}
		agent, err := sshTo(a, sshKey, "cybozu")
		if err != nil {
			time.Sleep(time.Second)
			goto RETRY
		}
		sshClients[a] = agent
	}

	return nil
}

func execAt(host string, args ...string) (stdout, stderr []byte, e error) {
	return execAtWithStream(host, nil, args...)
}

// WARNING: `input` can contain secret data.  Never output `input` to console.
func execAtWithInput(host string, input []byte, args ...string) (stdout, stderr []byte, e error) {
	var r io.Reader
	if input != nil {
		r = bytes.NewReader(input)
	}
	return execAtWithStream(host, r, args...)
}

// WARNING: `input` can contain secret data.  Never output `input` to console.
func execAtWithStream(host string, input io.Reader, args ...string) (stdout, stderr []byte, e error) {
	agent := sshClients[host]
	return doExec(agent, input, args...)
}

// WARNING: `input` can contain secret data.  Never output `input` to console.
func doExec(agent *sshAgent, input io.Reader, args ...string) ([]byte, []byte, error) {
	err := agent.conn.SetDeadline(time.Now().Add(DefaultRunTimeout))
	if err != nil {
		return nil, nil, err
	}
	defer agent.conn.SetDeadline(time.Time{})

	sess, err := agent.client.NewSession()
	if err != nil {
		return nil, nil, err
	}
	defer sess.Close()

	if input != nil {
		sess.Stdin = input
	}
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	sess.Stdout = outBuf
	sess.Stderr = errBuf
	err = sess.Run(strings.Join(args, " "))
	return outBuf.Bytes(), errBuf.Bytes(), err
}

func execSafeAt(host string, args ...string) []byte {
	stdout, stderr, err := execAt(host, args...)
	ExpectWithOffset(1, err).To(Succeed(), "[%s] %v: %s", host, args, stderr)
	return stdout
}

func execSafeGomegaAt(g Gomega, host string, args ...string) []byte {
	stdout, stderr, err := execAt(host, args...)
	g.ExpectWithOffset(1, err).To(Succeed(), "[%s] %v, stdout: %s, stderr: %s", host, args, stdout, stderr)
	return stdout
}

func execRetryAt(host string, handler retryHandler, args ...string) []byte {
	var stdout, stderr []byte
	var err error
	EventuallyWithOffset(1, func(g Gomega) {
		stdout, stderr, err = execAt(host, args...)
		if err != nil {
			msg := fmt.Sprintf("stdout: %s, stderr: %s, err: %v", string(stdout), string(stderr), err)
			if !handler(string(stdout), string(stderr), err) {
				StopTrying("retry skipped. " + msg).Wrap(err).Now()
			}
			fmt.Printf("retrying... %v", args)
			g.Expect(err).NotTo(HaveOccurred(), "retry failed. "+msg)
		}
	}).Should(Succeed())
	return stdout
}

// waitRequestComplete waits for the current request to be completed.
// If check is not "", the contents is also checked against the output from "neco status".
func waitRequestComplete(check string, recover ...bool) {
	// wait a moment for neco-updater to put a new request.
	time.Sleep(time.Second * 2)

	EventuallyWithOffset(1, func() error {
		stdout, stderr, err := execAt(bootServers[0], "neco", "status")
		if err != nil {
			return fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
		}
		out := string(stdout)

		// Sometimes, neco-worker aborts the update process. Detect it and recover if it is necessary.
		if strings.Contains(out, "status: aborted") {
			if len(recover) == 0 || !recover[0] {
				return StopTrying("update process is aborted: " + out)
			}
			fmt.Println(out)
			fmt.Println("update request is aborted, try to recover...")
			execAt(bootServers[0], "neco", "recover")
			return errors.New("update process is aborted: " + out)
		}

		if !strings.Contains(out, "status: completed") {
			return errors.New("request is not completed: " + out)
		}
		if check != "" && !strings.Contains(out, check) {
			return fmt.Errorf("should contain %q: "+out, check)
		}
		return nil

	}).Should(Succeed())
}

func getVaultToken() string {
	var token string
	Eventually(func() error {
		stdout, stderr, err := execAtWithInput(bootServers[0], []byte("cybozu"), "vault", "login",
			"-token-only", "-method=userpass", "username=admin", "password=-")
		if err != nil {
			return errors.New(string(stderr))
		}
		token = string(bytes.TrimSpace(stdout))
		return nil
	}).Should(Succeed())
	return token
}

func execEtcdctlAt(host string, args ...string) ([]byte, []byte, error) {
	execArgs := []string{"etcdctl", "--cert=/etc/neco/etcd.crt", "--key=/etc/neco/etcd.key"}
	execArgs = append(execArgs, args...)
	return execAt(host, execArgs...)
}

func checkDummyNetworkInterfacesOnBoot() {
	devices := []string{
		"node0",
		"bastion",
		"boot",
	}
	Eventually(func() error {
		for _, host := range bootServers {
			for _, netdev := range devices {
				_, _, err := execAt(host, "ip", "link", "show", netdev, "type", "dummy")
				if err != nil {
					return fmt.Errorf("%s is not exist on %s", netdev, host)
				}
			}
		}
		return nil
	}).Should(Succeed())
}

func checkSystemdServicesOnBoot() {
	services := []string{
		"bird.service",
		"systemd-networkd.service",
		"chrony.service",
	}
	Eventually(func() error {
		for _, host := range bootServers {
			for _, service := range services {
				_, _, err := execAt(host, "systemctl", "-q", "is-active", service)
				if err != nil {
					return fmt.Errorf("%s is not active on %s", service, host)
				}
			}
		}
		return nil
	}).Should(Succeed())
}

func getSerfBootMembers() (*serfMemberContainer, error) {
	stdout, stderr, err := execAt(bootServers[0], "serf", "members", "-format", "json", "-tag", "boot-server=true")
	if err != nil {
		return nil, fmt.Errorf("stdout=%s, stderr=%s err=%v", stdout, stderr, err)
	}
	var result serfMemberContainer
	err = json.Unmarshal(stdout, &result)
	if err != nil {
		return nil, fmt.Errorf("stdout=%s, stderr=%s err=%v", stdout, stderr, err)
	}
	return &result, nil
}

func getSerfWorkerMembers() (*serfMemberContainer, error) {
	stdout, stderr, err := execAt(bootServers[0], "serf", "members", "-format", "json", "-tag", "boot-server=false")
	if err != nil {
		return nil, fmt.Errorf("stdout=%s, stderr=%s err=%v", stdout, stderr, err)
	}
	var result serfMemberContainer
	err = json.Unmarshal(stdout, &result)
	if err != nil {
		return nil, fmt.Errorf("stdout=%s, stderr=%s err=%v", stdout, stderr, err)
	}
	return &result, nil
}
