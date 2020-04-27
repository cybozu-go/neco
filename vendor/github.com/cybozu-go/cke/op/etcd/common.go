package etcd

import (
	"context"
	"errors"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	"github.com/cybozu-go/cke/op/common"
)

func etcdEndpoints(nodes []*cke.Node) []string {
	endpoints := make([]string, len(nodes))
	for i, n := range nodes {
		endpoints[i] = "https://" + n.Address + ":2379"
	}
	return endpoints
}

func addressInURLs(address string, urls []string) (bool, error) {
	for _, urlStr := range urls {
		u, err := url.Parse(urlStr)
		if err != nil {
			return false, err
		}
		h, _, err := net.SplitHostPort(u.Host)
		if err != nil {
			return false, err
		}
		if h == address {
			return true, nil
		}
	}
	return false, nil
}

// BuiltInParams returns etcd parameters.
func BuiltInParams(node *cke.Node, initialCluster []string, state string) cke.ServiceParams {
	// NOTE: "--initial-*" flags and its value must be joined with '=' to
	// compare parameters to detect outdated parameters.
	args := []string{
		"--name=" + node.Address,
		"--listen-peer-urls=https://0.0.0.0:2380",
		"--listen-client-urls=https://0.0.0.0:2379",
		"--advertise-client-urls=https://" + node.Address + ":2379",
		"--cert-file=" + op.EtcdPKIPath("server.crt"),
		"--key-file=" + op.EtcdPKIPath("server.key"),
		"--client-cert-auth=true",
		"--trusted-ca-file=" + op.EtcdPKIPath("ca-client.crt"),
		"--peer-cert-file=" + op.EtcdPKIPath("peer.crt"),
		"--peer-key-file=" + op.EtcdPKIPath("peer.key"),
		"--peer-client-cert-auth=true",
		"--peer-trusted-ca-file=" + op.EtcdPKIPath("ca-peer.crt"),
		"--enable-v2=false",
		"--enable-pprof=true",
		"--auto-compaction-mode=periodic",
		// Mimic the default --etcd-compaction-interval of kube-apiserver (= 5 minutes).
		// https://kubernetes.io/docs/reference/command-line-tools-reference/kube-apiserver/
		// etcd version 3.3.3 and later will run compaction for every auto-compaction-retention
		// value if the value is less than 1 hour.
		// https://github.com/etcd-io/etcd/blob/master/Documentation/op-guide/maintenance.md#auto-compaction
		"--auto-compaction-retention=5m",
	}
	if len(initialCluster) > 0 {
		args = append(args,
			"--initial-advertise-peer-urls=https://"+node.Address+":2380",
			"--initial-cluster="+strings.Join(initialCluster, ","),
			"--initial-cluster-token=cke",
			"--initial-cluster-state="+state)
	}
	binds := []cke.Mount{
		{
			Source:      "/etc/etcd/pki",
			Destination: "/etc/etcd/pki",
			ReadOnly:    true,
			Label:       cke.LabelPrivate,
		},
	}
	params := cke.ServiceParams{
		ExtraArguments: args,
		ExtraBinds:     binds,
	}

	return params
}

type prepareEtcdCertificatesCommand struct {
	files  *common.FilesBuilder
	domain string
}

func (c prepareEtcdCertificatesCommand) Run(ctx context.Context, inf cke.Infrastructure, _ string) error {
	f := func(ctx context.Context, n *cke.Node) (cert, key []byte, err error) {
		c, k, e := cke.EtcdCA{}.IssueServerCert(ctx, inf, n, c.domain)
		if e != nil {
			return nil, nil, e
		}
		return []byte(c), []byte(k), nil
	}
	err := c.files.AddKeyPair(ctx, op.EtcdPKIPath("server"), f)
	if err != nil {
		return err
	}

	f = func(ctx context.Context, n *cke.Node) (cert, key []byte, err error) {
		c, k, e := cke.EtcdCA{}.IssuePeerCert(ctx, inf, n)
		if e != nil {
			return nil, nil, e
		}
		return []byte(c), []byte(k), nil
	}
	err = c.files.AddKeyPair(ctx, op.EtcdPKIPath("peer"), f)
	if err != nil {
		return err
	}

	peerCA, err := inf.Storage().GetCACertificate(ctx, "etcd-peer")
	if err != nil {
		return err
	}
	f2 := func(ctx context.Context, node *cke.Node) ([]byte, error) {
		return []byte(peerCA), nil
	}
	err = c.files.AddFile(ctx, op.EtcdPKIPath("ca-peer.crt"), f2)
	if err != nil {
		return err
	}

	clientCA, err := inf.Storage().GetCACertificate(ctx, "etcd-client")
	if err != nil {
		return err
	}
	f2 = func(ctx context.Context, node *cke.Node) ([]byte, error) {
		return []byte(clientCA), nil
	}
	err = c.files.AddFile(ctx, op.EtcdPKIPath("ca-client.crt"), f2)
	if err != nil {
		return err
	}
	return nil
}

func (c prepareEtcdCertificatesCommand) Command() cke.Command {
	return cke.Command{
		Name: "prepare-etcd-certificates",
	}
}

type waitEtcdSyncCommand struct {
	endpoints       []string
	checkRedundancy bool
}

func (c waitEtcdSyncCommand) try(ctx context.Context, inf cke.Infrastructure) error {
	cli, err := inf.NewEtcdClient(ctx, c.endpoints)
	if err != nil {
		return err
	}

	ct, cancel := context.WithTimeout(ctx, op.TimeoutDuration)
	defer cancel()
	resp, err := cli.Grant(ct, 10)
	if err != nil {
		return err
	}
	if resp.ID == clientv3.NoLease {
		return errors.New("no lease")
	}

	if !c.checkRedundancy {
		return nil
	}

	healthyMemberCount := 0
	for _, ep := range c.endpoints {
		ct2, cancel2 := context.WithTimeout(ctx, op.TimeoutDuration)
		_, err = cli.Status(ct2, ep)
		cancel2()
		if err == nil {
			healthyMemberCount++
		}
	}
	if healthyMemberCount <= int(len(c.endpoints)+1)/2 {
		return errors.New("etcd cluster is not redundant enough")
	}
	return nil
}

func (c waitEtcdSyncCommand) Run(ctx context.Context, inf cke.Infrastructure, _ string) error {
	for i := 0; i < 9; i++ {
		err := c.try(ctx, inf)
		if err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}

	// last try
	return c.try(ctx, inf)
}

func (c waitEtcdSyncCommand) Command() cke.Command {
	return cke.Command{
		Name:   "wait-etcd-sync",
		Target: strings.Join(c.endpoints, ","),
	}
}
