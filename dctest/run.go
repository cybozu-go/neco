package dctest

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
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

	proxy = "http://10.0.49.3:3128"
)

var (
	sshClients = make(map[string]*sshAgent)

	// https://github.com/cybozu-go/cke/pull/81/files
	agentDialer = &net.Dialer{
		Timeout:   defaultDialTimeout,
		KeepAlive: defaultKeepAlive,
	}
)

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

	data, err := ioutil.ReadAll(f)
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

// waitRequestComplete waits for the current request to be completed.
// If check is not "", the contents is also checked against the output from "neco status".
func waitRequestComplete(check string) {
	// wait a moment for neco-updater to put a new request.
	time.Sleep(time.Second * 2)

	EventuallyWithOffset(1, func() error {
		stdout, _, err := execAt(boot0, "neco", "status")
		if err != nil {
			return err
		}
		out := string(stdout)
		if !strings.Contains(out, "status: completed") {
			return errors.New("request is not completed: " + out)
		}
		if check != "" && !strings.Contains(out, check) {
			return errors.New("request is not completed: " + out)
		}
		return nil

	}).Should(Succeed())
}

func getVaultToken() string {
	stdout, stderr, err := execAtWithInput(boot0, []byte("cybozu"), "vault", "login",
		"-token-only", "-method=userpass", "username=admin", "password=-")
	Expect(err).ShouldNot(HaveOccurred(), "stderr=%s", stderr)

	return string(bytes.TrimSpace(stdout))
}
