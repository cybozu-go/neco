package etcd

import (
	"context"
	"strings"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	"github.com/cybozu-go/cke/op/common"
)

type bootOp struct {
	endpoints []string
	nodes     []*cke.Node
	params    cke.EtcdParams
	step      int
	files     *common.FilesBuilder
	domain    string
}

// BootOp returns an Operator to bootstrap etcd cluster.
func BootOp(nodes []*cke.Node, params cke.EtcdParams, domain string) cke.Operator {
	return &bootOp{
		endpoints: etcdEndpoints(nodes),
		nodes:     nodes,
		params:    params,
		files:     common.NewFilesBuilder(nodes),
		domain:    domain,
	}
}

func (o *bootOp) Name() string {
	return "etcd-bootstrap"
}

func (o *bootOp) NextCommand() cke.Commander {
	volname := op.EtcdVolumeName(o.params)

	switch o.step {
	case 0:
		o.step++
		return common.ImagePullCommand(o.nodes, cke.EtcdImage)
	case 1:
		o.step++
		return prepareEtcdCertificatesCommand{o.files, o.domain}
	case 2:
		o.step++
		return o.files
	case 3:
		o.step++
		return common.VolumeCreateCommand(o.nodes, volname)
	case 4:
		o.step++
		opts := []string{
			"--mount",
			"type=volume,src=" + volname + ",dst=/var/lib/etcd",
		}
		initialCluster := make([]string, len(o.nodes))
		for i, n := range o.nodes {
			initialCluster[i] = n.Address + "=https://" + n.Address + ":2380"
		}
		paramsMap := make(map[string]cke.ServiceParams)
		for _, n := range o.nodes {
			paramsMap[n.Address] = BuiltInParams(n, initialCluster, "new")
		}
		return common.RunContainerCommand(o.nodes, op.EtcdContainerName, cke.EtcdImage,
			common.WithOpts(opts),
			common.WithParamsMap(paramsMap),
			common.WithExtra(o.params.ServiceParams))
	case 5:
		o.step++
		return waitEtcdSyncCommand{o.endpoints, false}
	case 6:
		o.step++
		return setupEtcdAuthCommand{o.endpoints}
	case 7:
		o.step++
		return common.VolumeCreateCommand(o.nodes, op.EtcdAddedMemberVolumeName)
	default:
		return nil
	}
}

func (o *bootOp) Targets() []string {
	ips := make([]string, len(o.nodes))
	for i, n := range o.nodes {
		ips[i] = n.Address
	}
	return ips
}

type setupEtcdAuthCommand struct {
	endpoints []string
}

func (c setupEtcdAuthCommand) Run(ctx context.Context, inf cke.Infrastructure, _ string) error {
	cli, err := inf.NewEtcdClient(ctx, c.endpoints)
	if err != nil {
		return err
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	err = cke.AddUserRole(ctx, cli, "root", "")
	if err != nil {
		return err
	}
	_, err = cli.UserGrantRole(ctx, "root", "root")
	if err != nil {
		return err
	}

	err = cke.AddUserRole(ctx, cli, "kube-apiserver", "/registry/")
	if err != nil {
		return err
	}

	_, err = cli.AuthEnable(ctx)
	return err
}

func (c setupEtcdAuthCommand) Command() cke.Command {
	return cke.Command{
		Name:   "setup-etcd-auth",
		Target: strings.Join(c.endpoints, ","),
	}
}
