package cke

import (
	"bytes"
	"net"
	"strings"
	"time"

	"github.com/cybozu-go/log"
	"golang.org/x/crypto/ssh"
)

const (
	defaultDialTimeout = 30 * time.Second
	defaultKeepAlive   = 5 * time.Second

	// DefaultRunTimeout is the timeout value for Agent.Run().
	DefaultRunTimeout = 10 * time.Minute
)

var (
	// When KeepAlive is > 0, the dialer returns TCP connections
	// with keep-alive enabled.  With the default 5 second duration,
	// the implementation can detect dead peer in around 50 seconds.
	agentDialer = &net.Dialer{
		Timeout:   defaultDialTimeout,
		KeepAlive: defaultKeepAlive,
	}
)

// Agent is the interface to run commands on a node.
type Agent interface {
	// Close closes the underlying connection.
	Close() error

	// Run command on the node.
	// It returns non-nil error if the command takes too long (> DefaultRunTimeout).
	Run(command string) (stdout, stderr []byte, err error)

	// RunWithInput run command with input as stdin.
	// It returns non-nil error if the command takes too long (> DefaultRunTimeout).
	RunWithInput(command, input string) error

	// RunWithTimeout run command with given timeout.
	// If timeout is 0, the command will run indefinitely.
	RunWithTimeout(command, input string, timeout time.Duration) (stdout, stderr []byte, err error)
}

type sshAgent struct {
	node   *Node
	client *ssh.Client
	conn   net.Conn
}

// SSHAgent creates an Agent that communicates over SSH.
// It returns non-nil error when connection could not be established.
func SSHAgent(node *Node, privkey string) (Agent, error) {
	conn, err := agentDialer.Dial("tcp", node.Address+":22")
	if err != nil {
		log.Error("failed to dial: ", map[string]interface{}{
			log.FnError: err,
			"address":   node.Address,
		})
		return nil, err
	}
	signer, err := ssh.ParsePrivateKey([]byte(privkey))
	if err != nil {
		return nil, err
	}

	config := &ssh.ClientConfig{
		User: node.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
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
		node:   node,
		client: ssh.NewClient(clientConn, channelCh, reqCh),
		conn:   conn,
	}
	_, _, err = a.Run("docker version")
	if err != nil {
		a.Close()
		return nil, err
	}

	return a, nil
}

func (a sshAgent) Close() error {
	err := a.client.Close()
	a.client = nil
	return err
}

func (a sshAgent) Run(command string) ([]byte, []byte, error) {
	return a.RunWithTimeout(command, "", DefaultRunTimeout)
}

func (a sshAgent) RunWithInput(command, input string) error {
	_, _, err := a.RunWithTimeout(command, input, DefaultRunTimeout)
	return err
}

func (a sshAgent) RunWithTimeout(command, input string, timeout time.Duration) ([]byte, []byte, error) {
	if timeout > 0 {
		err := a.conn.SetDeadline(time.Now().Add(timeout))
		if err != nil {
			return nil, nil, err
		}

		defer a.conn.SetDeadline(time.Time{})
	}

	session, err := a.client.NewSession()
	if err != nil {
		log.Error("failed to create session: ", map[string]interface{}{
			log.FnError: err,
		})
		return nil, nil, err
	}
	defer session.Close()

	if len(input) > 0 {
		session.Stdin = strings.NewReader(input)
	}

	var stdoutBuff bytes.Buffer
	var stderrBuff bytes.Buffer
	session.Stdout = &stdoutBuff
	session.Stderr = &stderrBuff
	err = session.Run(command)
	stdout := stdoutBuff.Bytes()
	stderr := stderrBuff.Bytes()
	if err != nil {
		log.Error("failed to run command: ", map[string]interface{}{
			log.FnError: err,
			"command":   command,
			"stderr":    string(stderr),
		})
		return stdout, stderr, err
	}
	return stdout, stderr, nil
}
