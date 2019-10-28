package server

import (
	"context"
	"fmt"
	"sync"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
)

// GetClusterStatus consults the whole cluster and constructs *ClusterStatus.
func (c Controller) GetClusterStatus(ctx context.Context, cluster *cke.Cluster, inf cke.Infrastructure) (*cke.ClusterStatus, error) {
	var mu sync.Mutex
	statuses := make(map[string]*cke.NodeStatus)

	env := well.NewEnvironment(ctx)
	for _, n := range cluster.Nodes {
		n := n
		env.Go(func(ctx context.Context) error {
			ns, err := op.GetNodeStatus(ctx, inf, n, cluster)
			if err != nil {
				return fmt.Errorf("%s: %v", n.Address, err)
			}

			mu.Lock()
			statuses[n.Address] = ns
			mu.Unlock()
			return nil
		})
	}
	env.Stop()
	err := env.Wait()
	if err != nil {
		return nil, err
	}

	cs := new(cke.ClusterStatus)
	cs.NodeStatuses = statuses

	var etcdRunning bool
	for _, n := range cke.ControlPlanes(cluster.Nodes) {
		ns := statuses[n.Address]
		if ns.Etcd.HasData {
			etcdRunning = true
			break
		}
	}

	if etcdRunning {
		ecs, err := op.GetEtcdClusterStatus(ctx, inf, cluster.Nodes)
		if err != nil {
			log.Warn("failed to get etcd cluster status", map[string]interface{}{
				log.FnError: err,
			})
			// return nil
			return cs, nil
		}
		cs.Etcd = ecs
	}

	var livingMaster *cke.Node
	for _, n := range cke.ControlPlanes(cluster.Nodes) {
		ns := statuses[n.Address]
		if ns.APIServer.Running {
			livingMaster = n
			break
		}
	}

	if livingMaster != nil {
		cs.Kubernetes, err = op.GetKubernetesClusterStatus(ctx, inf, livingMaster, cluster)
		if err != nil {
			log.Error("failed to get kubernetes cluster status", map[string]interface{}{
				log.FnError: err,
			})
			return nil, err
		}
	}
	return cs, nil
}
