package etcd

import (
	"context"
	"fmt"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	"github.com/cybozu-go/cke/op/common"
	"github.com/cybozu-go/log"
)

type addMemberOp struct {
	endpoints  []string
	targetNode *cke.Node
	params     cke.EtcdParams
	step       int
	files      *common.FilesBuilder
	domain     string
}

// AddMemberOp returns an Operator to add member to etcd cluster.
func AddMemberOp(cp []*cke.Node, targetNode *cke.Node, params cke.EtcdParams, domain string) cke.Operator {
	return &addMemberOp{
		endpoints:  etcdEndpoints(cp),
		targetNode: targetNode,
		params:     params,
		files:      common.NewFilesBuilder([]*cke.Node{targetNode}),
		domain:     domain,
	}
}

func (o *addMemberOp) Name() string {
	return "etcd-add-member"
}

func (o *addMemberOp) NextCommand() cke.Commander {
	volname := op.EtcdVolumeName(o.params)
	extra := o.params.ServiceParams

	nodes := []*cke.Node{o.targetNode}
	switch o.step {
	case 0:
		o.step++
		return common.ImagePullCommand(nodes, cke.EtcdImage)
	case 1:
		o.step++
		return common.StopContainerCommand(o.targetNode, op.EtcdContainerName)
	case 2:
		o.step++
		return common.VolumeRemoveCommand(nodes, op.EtcdAddedMemberVolumeName)
	case 3:
		o.step++
		return common.VolumeRemoveCommand(nodes, volname)
	case 4:
		o.step++
		return common.VolumeCreateCommand(nodes, volname)
	case 5:
		o.step++
		return prepareEtcdCertificatesCommand{o.files, o.domain}
	case 6:
		o.step++
		return o.files
	case 7:
		o.step++
		opts := []string{
			"--mount",
			"type=volume,src=" + volname + ",dst=/var/lib/etcd",
		}
		return addMemberCommand{o.endpoints, o.targetNode, opts, extra}
	case 8:
		o.step++
		return waitEtcdSyncCommand{etcdEndpoints([]*cke.Node{o.targetNode}), false}
	case 9:
		o.step++
		return common.VolumeCreateCommand(nodes, op.EtcdAddedMemberVolumeName)
	}
	return nil
}

func (o *addMemberOp) Targets() []string {
	return []string{
		o.targetNode.Address,
	}
}

type addMemberCommand struct {
	endpoints []string
	node      *cke.Node
	opts      []string
	extra     cke.ServiceParams
}

func (c addMemberCommand) Run(ctx context.Context, inf cke.Infrastructure, _ string) error {
	cli, err := inf.NewEtcdClient(ctx, c.endpoints)
	if err != nil {
		return err
	}
	defer cli.Close()

	ct, cancel := context.WithTimeout(ctx, op.TimeoutDuration)
	defer cancel()
	resp, err := cli.MemberList(ct)
	if err != nil {
		return err
	}
	members := resp.Members

	inMember := false
	for _, m := range members {
		inMember, err = addressInURLs(c.node.Address, m.PeerURLs)
		if err != nil {
			return err
		}
		if inMember {
			break
		}
	}

	if !inMember {
		// wait for several seconds to satisfy etcd server check
		// https://github.com/etcd-io/etcd/blob/fb674833c21e729fe87fff4addcf93b2aa4df9df/etcdserver/server.go#L1562
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(10 * time.Second):
		}

		ct, cancel := context.WithTimeout(ctx, op.TimeoutDuration)
		defer cancel()
		resp, err := cli.MemberAdd(ct, []string{fmt.Sprintf("https://%s:2380", c.node.Address)})
		if err != nil {
			return err
		}
		members = resp.Members
	}
	log.Debug("retrieved memgers from etcd", map[string]interface{}{
		"members": members,
	})

	// gofail: var etcdAfterMemberAdd struct{}
	ce := inf.Engine(c.node.Address)
	ss, err := ce.Inspect([]string{op.EtcdContainerName})
	if err != nil {
		return err
	}
	if ss[op.EtcdContainerName].Running {
		return nil
	}

	var initialCluster []string
	for _, m := range members {
		for _, u := range m.PeerURLs {
			if len(m.Name) == 0 {
				initialCluster = append(initialCluster, c.node.Address+"="+u)
			} else {
				initialCluster = append(initialCluster, m.Name+"="+u)
			}
		}
	}

	return ce.RunSystem(op.EtcdContainerName, cke.EtcdImage, c.opts, BuiltInParams(c.node, initialCluster, "existing"), c.extra)
}

func (c addMemberCommand) Command() cke.Command {
	return cke.Command{
		Name: "add-etcd-member",
	}
}
