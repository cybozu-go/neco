package server

import (
	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	"github.com/cybozu-go/cke/op/clusterdns"
	"github.com/cybozu-go/cke/op/etcd"
	"github.com/cybozu-go/cke/op/etcdbackup"
	"github.com/cybozu-go/cke/op/k8s"
	"github.com/cybozu-go/cke/op/nodedns"
	"github.com/cybozu-go/cke/static"
	"github.com/cybozu-go/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DecideOps returns the next operations to do and the operation phase.
// This returns nil when no operations need to be done.
func DecideOps(c *cke.Cluster, cs *cke.ClusterStatus, resources []cke.ResourceDefinition) ([]cke.Operator, cke.OperationPhase) {
	nf := NewNodeFilter(c, cs)

	// 0. Execute upgrade operation if necessary
	if cs.ConfigVersion != cke.ConfigVersion {
		// Upgrade operations run only when all CPs are SSH reachable
		if len(nf.SSHNotConnectedNodes(nf.cluster.Nodes, true, false)) > 0 {
			log.Warn("cannot upgrade for unreachable nodes", nil)
			return nil, cke.PhaseUpgradeAborted
		}
		return []cke.Operator{op.UpgradeOp(cs.ConfigVersion, nf.ControlPlane())}, cke.PhaseUpgrade
	}

	// 1. Run or restart rivers.  This guarantees:
	// - CKE tools image is pulled on all nodes.
	// - Rivers runs on all nodes and will proxy requests only to control plane nodes.
	if ops := riversOps(c, nf); len(ops) > 0 {
		return ops, cke.PhaseRivers
	}

	// 2. Bootstrap etcd cluster, if not yet.
	if !nf.EtcdBootstrapped() {
		// Etcd boot operations run only when all CPs are SSH reachable
		if len(nf.SSHNotConnectedNodes(nf.cluster.Nodes, true, false)) > 0 {
			log.Warn("cannot bootstrap etcd for unreachable nodes", nil)
			return nil, cke.PhaseEtcdBootAborted
		}
		return []cke.Operator{etcd.BootOp(nf.ControlPlane(), c.Options.Etcd, c.Options.Kubelet.Domain)}, cke.PhaseEtcdBoot
	}

	// 3. Start etcd containers.
	if nodes := nf.SSHConnectedNodes(nf.EtcdStoppedMembers(), true, false); len(nodes) > 0 {
		return []cke.Operator{etcd.StartOp(nodes, c.Options.Etcd, c.Options.Kubelet.Domain)}, cke.PhaseEtcdStart
	}

	// 4. Wait for etcd cluster to become ready
	if !cs.Etcd.IsHealthy {
		return []cke.Operator{etcd.WaitClusterOp(nf.ControlPlane())}, cke.PhaseEtcdWait
	}

	// 5. Run or restart kubernetes components.
	if ops := k8sOps(c, nf); len(ops) > 0 {
		return ops, cke.PhaseK8sStart
	}

	// 6. Maintain etcd cluster, only when all CPs are SSH reachable.
	if len(nf.SSHNotConnectedNodes(nf.cluster.Nodes, true, false)) == 0 {
		if o := etcdMaintOp(c, nf); o != nil {
			return []cke.Operator{o}, cke.PhaseEtcdMaintain
		}
	}

	// 7. Maintain k8s resources.
	if ops := k8sMaintOps(c, cs, resources, nf); len(ops) > 0 {
		return ops, cke.PhaseK8sMaintain
	}

	// 8. Stop and delete control plane services running on non control plane nodes.
	if ops := cleanOps(c, nf); len(ops) > 0 {
		return ops, cke.PhaseStopCP
	}

	return nil, cke.PhaseCompleted
}

func riversOps(c *cke.Cluster, nf *NodeFilter) (ops []cke.Operator) {
	if nodes := nf.SSHConnectedNodes(nf.RiversStoppedNodes(), true, true); len(nodes) > 0 {
		ops = append(ops, op.RiversBootOp(nodes, nf.ControlPlane(), c.Options.Rivers, op.RiversContainerName, op.RiversUpstreamPort, op.RiversListenPort))
	}
	if nodes := nf.SSHConnectedNodes(nf.RiversOutdatedNodes(), true, true); len(nodes) > 0 {
		ops = append(ops, op.RiversRestartOp(nodes, nf.ControlPlane(), c.Options.Rivers, op.RiversContainerName, op.RiversUpstreamPort, op.RiversListenPort))
	}
	if nodes := nf.SSHConnectedNodes(nf.EtcdRiversStoppedNodes(), true, false); len(nodes) > 0 {
		ops = append(ops, op.RiversBootOp(nodes, nf.ControlPlane(), c.Options.EtcdRivers, op.EtcdRiversContainerName, op.EtcdRiversUpstreamPort, op.EtcdRiversListenPort))
	}
	if nodes := nf.SSHConnectedNodes(nf.EtcdRiversOutdatedNodes(), true, false); len(nodes) > 0 {
		ops = append(ops, op.RiversRestartOp(nodes, nf.ControlPlane(), c.Options.EtcdRivers, op.EtcdRiversContainerName, op.EtcdRiversUpstreamPort, op.EtcdRiversListenPort))
	}
	return ops
}

func k8sOps(c *cke.Cluster, nf *NodeFilter) (ops []cke.Operator) {
	// For cp nodes
	if nodes := nf.SSHConnectedNodes(nf.APIServerStoppedNodes(), true, false); len(nodes) > 0 {
		ops = append(ops, k8s.APIServerBootOp(nodes, nf.ControlPlane(), c.ServiceSubnet, c.Options.Kubelet.Domain, c.Options.APIServer))
	}
	if nodes := nf.SSHConnectedNodes(nf.APIServerOutdatedNodes(), true, false); len(nodes) > 0 {
		ops = append(ops, k8s.APIServerRestartOp(nodes, nf.ControlPlane(), c.ServiceSubnet, c.Options.Kubelet.Domain, c.Options.APIServer))
	}
	if nodes := nf.SSHConnectedNodes(nf.ControllerManagerStoppedNodes(), true, false); len(nodes) > 0 {
		ops = append(ops, k8s.ControllerManagerBootOp(nodes, c.Name, c.ServiceSubnet, c.Options.ControllerManager))
	}
	if nodes := nf.SSHConnectedNodes(nf.ControllerManagerOutdatedNodes(), true, false); len(nodes) > 0 {
		ops = append(ops, k8s.ControllerManagerRestartOp(nodes, c.Name, c.ServiceSubnet, c.Options.ControllerManager))
	}
	if nodes := nf.SSHConnectedNodes(nf.SchedulerStoppedNodes(), true, false); len(nodes) > 0 {
		ops = append(ops, k8s.SchedulerBootOp(nodes, c.Name, c.Options.Scheduler))
	}
	if nodes := nf.SSHConnectedNodes(nf.SchedulerOutdatedNodes(c.Options.Scheduler), true, false); len(nodes) > 0 {
		ops = append(ops, k8s.SchedulerRestartOp(nodes, c.Name, c.Options.Scheduler))
	}

	// For all nodes
	apiServer := nf.HealthyAPIServer()
	if nodes := nf.SSHConnectedNodes(nf.KubeletUnrecognizedNodes(), true, true); len(nodes) > 0 {
		ops = append(ops, k8s.KubeletRestartOp(nodes, c.Name, c.Options.Kubelet))
	}
	if nodes := nf.SSHConnectedNodes(nf.KubeletStoppedNodes(), true, true); len(nodes) > 0 {
		ops = append(ops, k8s.KubeletBootOp(nodes, nf.KubeletStoppedRegisteredNodes(), apiServer, c.Name, c.Options.Kubelet))
	}
	if nodes := nf.SSHConnectedNodes(nf.KubeletOutdatedNodes(), true, true); len(nodes) > 0 {
		ops = append(ops, k8s.KubeletRestartOp(nodes, c.Name, c.Options.Kubelet))
	}
	nupvNodes, nupvs := nf.NeedUpdateUpBlockPVsToV1_16()
	if nodes := nf.SSHConnectedNodes(nupvNodes, true, true); len(nodes) > 0 {
		ops = append(ops, k8s.UpdateBlockPVsUpToV1_16Op(apiServer, nodes, nupvs))
	}
	if nodes := nf.SSHConnectedNodes(nf.ProxyStoppedNodes(), true, true); len(nodes) > 0 {
		ops = append(ops, k8s.KubeProxyBootOp(nodes, c.Name, c.Options.Proxy))
	}
	if nodes := nf.SSHConnectedNodes(nf.ProxyOutdatedNodes(), true, true); len(nodes) > 0 {
		ops = append(ops, k8s.KubeProxyRestartOp(nodes, c.Name, c.Options.Proxy))
	}
	return ops
}

func etcdMaintOp(c *cke.Cluster, nf *NodeFilter) cke.Operator {
	// this function is called only when all the CPs are reachable.
	// so, filtering by SSHConnectedNodes(nodes, true, ...) is not required.

	if members := nf.EtcdNonClusterMembers(false); len(members) > 0 {
		return etcd.RemoveMemberOp(nf.ControlPlane(), members)
	}
	if nodes, ids := nf.EtcdNonCPMembers(false); len(nodes) > 0 {
		return etcd.DestroyMemberOp(nf.ControlPlane(), nf.SSHConnectedNodes(nodes, false, true), ids)
	}
	if nodes := nf.EtcdUnstartedMembers(); len(nodes) > 0 {
		return etcd.AddMemberOp(nf.ControlPlane(), nodes[0], c.Options.Etcd, c.Options.Kubelet.Domain)
	}

	if !nf.EtcdIsGood() {
		log.Warn("etcd is not good for maintenance", nil)
		// return nil to proceed to k8s maintenance.
		return nil
	}

	// Adding members or removing/restarting healthy members is done only when
	// all members are in sync.

	if nodes := nf.EtcdNewMembers(); len(nodes) > 0 {
		return etcd.AddMemberOp(nf.ControlPlane(), nodes[0], c.Options.Etcd, c.Options.Kubelet.Domain)
	}
	if members := nf.EtcdNonClusterMembers(true); len(members) > 0 {
		return etcd.RemoveMemberOp(nf.ControlPlane(), members)
	}
	if nodes, ids := nf.EtcdNonCPMembers(true); len(nodes) > 0 {
		return etcd.DestroyMemberOp(nf.ControlPlane(), nf.SSHConnectedNodes(nodes, false, true), ids)
	}
	if nodes := nf.EtcdOutdatedMembers(); len(nodes) > 0 {
		return etcd.RestartOp(nf.ControlPlane(), nodes[0], c.Options.Etcd)
	}

	return nil
}

func k8sMaintOps(c *cke.Cluster, cs *cke.ClusterStatus, resources []cke.ResourceDefinition, nf *NodeFilter) (ops []cke.Operator) {
	ks := cs.Kubernetes
	apiServer := nf.HealthyAPIServer()

	if !ks.IsControlPlaneReady {
		return []cke.Operator{op.KubeWaitOp(apiServer)}
	}

	ops = append(ops, decideResourceOps(apiServer, ks, resources, ks.IsReady(c))...)

	ops = append(ops, decideClusterDNSOps(apiServer, c, ks)...)

	ops = append(ops, decideNodeDNSOps(apiServer, c, ks)...)

	var cpReadyAddresses []corev1.EndpointAddress
	for _, n := range nf.HealthyAPIServerNodes() {
		cpReadyAddresses = append(cpReadyAddresses, corev1.EndpointAddress{
			IP: n.Address,
		})
	}
	var cpNotReadyAddresses []corev1.EndpointAddress
	for _, n := range nf.UnhealthyAPIServerNodes() {
		cpNotReadyAddresses = append(cpNotReadyAddresses, corev1.EndpointAddress{
			IP: n.Address,
		})
	}

	masterEP := &corev1.Endpoints{}
	masterEP.Namespace = metav1.NamespaceDefault
	masterEP.Name = "kubernetes"
	masterEP.Subsets = []corev1.EndpointSubset{
		{
			Addresses:         cpReadyAddresses,
			NotReadyAddresses: cpNotReadyAddresses,
			Ports: []corev1.EndpointPort{
				{
					Name:     "https",
					Port:     6443,
					Protocol: corev1.ProtocolTCP,
				},
			},
		},
	}
	epOp := decideEpOp(masterEP, ks.MasterEndpoints, apiServer)
	if epOp != nil {
		ops = append(ops, epOp)
	}

	// Endpoints needs a corresponding Service.
	// If an Endpoints lacks such a Service, it will be removed.
	// https://github.com/kubernetes/kubernetes/blob/b7c2d923ef4e166b9572d3aa09ca72231b59b28b/pkg/controller/endpoint/endpoints_controller.go#L392-L397
	svcOp := decideEtcdServiceOps(apiServer, ks.EtcdService)
	if svcOp != nil {
		ops = append(ops, svcOp)
	}

	cpAddresses := make([]corev1.EndpointAddress, len(nf.ControlPlane()))
	for i, cp := range nf.ControlPlane() {
		cpAddresses[i] = corev1.EndpointAddress{
			IP: cp.Address,
		}
	}
	etcdEP := &corev1.Endpoints{}
	etcdEP.Namespace = metav1.NamespaceSystem
	etcdEP.Name = op.EtcdEndpointsName
	etcdEP.Subsets = []corev1.EndpointSubset{
		{
			Addresses: cpAddresses,
			Ports: []corev1.EndpointPort{
				{
					Port:     2379,
					Protocol: corev1.ProtocolTCP,
				},
			},
		},
	}
	epOp = decideEpOp(etcdEP, ks.EtcdEndpoints, apiServer)
	if epOp != nil {
		ops = append(ops, epOp)
	}

	if nodes := nf.OutdatedAttrsNodes(); len(nodes) > 0 {
		ops = append(ops, op.KubeNodeUpdateOp(apiServer, nodes))
	}

	if nodes := nf.NonClusterNodes(); len(nodes) > 0 {
		ops = append(ops, op.KubeNodeRemoveOp(apiServer, nodes))
	}

	ops = append(ops, decideEtcdBackupOps(apiServer, c, ks)...)

	return ops
}

func decideClusterDNSOps(apiServer *cke.Node, c *cke.Cluster, ks cke.KubernetesClusterStatus) (ops []cke.Operator) {
	desiredDNSServers := c.DNSServers
	if ks.DNSService != nil {
		switch ip := ks.DNSService.Spec.ClusterIP; ip {
		case "", "None":
		default:
			desiredDNSServers = []string{ip}
		}
	}
	desiredClusterDomain := c.Options.Kubelet.Domain

	if len(desiredClusterDomain) == 0 {
		panic("Options.Kubelet.Domain is empty")
	}

	if ks.ClusterDNS.ConfigMap == nil {
		ops = append(ops, clusterdns.CreateConfigMapOp(apiServer, desiredClusterDomain, desiredDNSServers))
	} else {
		actualConfigData := ks.ClusterDNS.ConfigMap.Data
		expectedConfig := clusterdns.ConfigMap(desiredClusterDomain, desiredDNSServers)
		if actualConfigData["Corefile"] != expectedConfig.Data["Corefile"] {
			ops = append(ops, clusterdns.UpdateConfigMapOp(apiServer, expectedConfig))
		}
	}

	return ops
}

func decideNodeDNSOps(apiServer *cke.Node, c *cke.Cluster, ks cke.KubernetesClusterStatus) (ops []cke.Operator) {
	if len(ks.ClusterDNS.ClusterIP) == 0 {
		return nil
	}

	desiredDNSServers := c.DNSServers
	if ks.DNSService != nil {
		switch ip := ks.DNSService.Spec.ClusterIP; ip {
		case "", "None":
		default:
			desiredDNSServers = []string{ip}
		}
	}

	if ks.NodeDNS.ConfigMap == nil {
		ops = append(ops, nodedns.CreateConfigMapOp(apiServer, ks.ClusterDNS.ClusterIP, c.Options.Kubelet.Domain, desiredDNSServers))
	} else {
		actualConfigData := ks.NodeDNS.ConfigMap.Data
		expectedConfig := nodedns.ConfigMap(ks.ClusterDNS.ClusterIP, c.Options.Kubelet.Domain, desiredDNSServers)
		if actualConfigData["unbound.conf"] != expectedConfig.Data["unbound.conf"] {
			ops = append(ops, nodedns.UpdateConfigMapOp(apiServer, expectedConfig))
		}
	}

	return ops
}

func decideEpOp(expect, actual *corev1.Endpoints, apiServer *cke.Node) cke.Operator {
	if actual == nil {
		return op.KubeEndpointsCreateOp(apiServer, expect)
	}

	updateOp := op.KubeEndpointsUpdateOp(apiServer, expect)
	if len(actual.Subsets) != 1 {
		return updateOp
	}

	subset := actual.Subsets[0]
	if len(subset.Ports) != 1 || subset.Ports[0].Port != expect.Subsets[0].Ports[0].Port {
		return updateOp
	}

	if len(subset.Addresses) != len(expect.Subsets[0].Addresses) || len(subset.NotReadyAddresses) != len(expect.Subsets[0].NotReadyAddresses) {
		return updateOp
	}

	endpoints := make(map[string]bool)
	for _, a := range expect.Subsets[0].Addresses {
		endpoints[a.IP] = true
	}
	for _, a := range subset.Addresses {
		if !endpoints[a.IP] {
			return updateOp
		}
	}

	endpoints = make(map[string]bool)
	for _, a := range expect.Subsets[0].NotReadyAddresses {
		endpoints[a.IP] = true
	}
	for _, a := range subset.NotReadyAddresses {
		if !endpoints[a.IP] {
			return updateOp
		}
	}

	return nil
}

func decideEtcdServiceOps(apiServer *cke.Node, svc *corev1.Service) cke.Operator {
	if svc == nil {
		return op.KubeEtcdServiceCreateOp(apiServer)
	}

	updateOp := op.KubeEtcdServiceUpdateOp(apiServer)

	if len(svc.Spec.Ports) != 1 {
		return updateOp
	}
	if svc.Spec.Ports[0].Port != 2379 {
		return updateOp
	}
	if svc.Spec.Type != corev1.ServiceTypeClusterIP {
		return updateOp
	}
	if svc.Spec.ClusterIP != corev1.ClusterIPNone {
		return updateOp
	}

	return nil
}

func decideEtcdBackupOps(apiServer *cke.Node, c *cke.Cluster, ks cke.KubernetesClusterStatus) (ops []cke.Operator) {
	if c.EtcdBackup.Enabled == false {
		if ks.EtcdBackup.ConfigMap != nil {
			ops = append(ops, etcdbackup.ConfigMapRemoveOp(apiServer))
		}
		if ks.EtcdBackup.Secret != nil {
			ops = append(ops, etcdbackup.SecretRemoveOp(apiServer))
		}
		if ks.EtcdBackup.CronJob != nil {
			ops = append(ops, etcdbackup.CronJobRemoveOp(apiServer))
		}
		if ks.EtcdBackup.Service != nil {
			ops = append(ops, etcdbackup.ServiceRemoveOp(apiServer))
		}
		if ks.EtcdBackup.Pod != nil {
			ops = append(ops, etcdbackup.PodRemoveOp(apiServer))
		}
		return ops
	}

	if ks.EtcdBackup.ConfigMap == nil {
		ops = append(ops, etcdbackup.ConfigMapCreateOp(apiServer, c.EtcdBackup.Rotate))
	} else {
		actual := ks.EtcdBackup.ConfigMap.Data["config.yml"]
		expected := etcdbackup.RenderConfigMap(c.EtcdBackup.Rotate).Data["config.yml"]
		if actual != expected {
			ops = append(ops, etcdbackup.ConfigMapUpdateOp(apiServer, c.EtcdBackup.Rotate))
		}
	}
	if ks.EtcdBackup.Secret == nil {
		ops = append(ops, etcdbackup.SecretCreateOp(apiServer))
	}
	if ks.EtcdBackup.Service == nil {
		ops = append(ops, etcdbackup.ServiceCreateOp(apiServer))
	}
	if ks.EtcdBackup.Pod == nil {
		ops = append(ops, etcdbackup.PodCreateOp(apiServer, c.EtcdBackup.PVCName))
	} else if needUpdateEtcdBackupPod(c, ks) {
		ops = append(ops, etcdbackup.PodUpdateOp(apiServer, c.EtcdBackup.PVCName))
	}

	if ks.EtcdBackup.CronJob == nil {
		ops = append(ops, etcdbackup.CronJobCreateOp(apiServer, c.EtcdBackup.Schedule))
	} else if ks.EtcdBackup.CronJob.Spec.Schedule != c.EtcdBackup.Schedule {
		ops = append(ops, etcdbackup.CronJobUpdateOp(apiServer, c.EtcdBackup.Schedule))
	}

	return ops
}

func needUpdateEtcdBackupPod(c *cke.Cluster, ks cke.KubernetesClusterStatus) bool {
	volumes := ks.EtcdBackup.Pod.Spec.Volumes
	vol := new(corev1.Volume)
	for _, v := range volumes {
		if v.Name == "etcdbackup" {
			vol = &v
			break
		}
	}
	if vol == nil {
		return true
	}

	if vol.PersistentVolumeClaim == nil {
		return true
	}
	if vol.PersistentVolumeClaim.ClaimName != c.EtcdBackup.PVCName {
		return true
	}
	return false
}

func decideResourceOps(apiServer *cke.Node, ks cke.KubernetesClusterStatus, resources []cke.ResourceDefinition, isReady bool) (ops []cke.Operator) {
	for _, res := range static.Resources {
		// To avoid thundering herd problem. Deployments need to be created only after enough nodes become ready.
		if res.Kind == cke.KindDeployment && !isReady {
			continue
		}
		status, ok := ks.ResourceStatuses[res.Key]
		if !ok || res.NeedUpdate(&status) {
			ops = append(ops, op.ResourceApplyOp(apiServer, res, !status.HasBeenSSA))
		}
	}
	for _, res := range resources {
		if res.Kind == cke.KindDeployment && !isReady {
			continue
		}
		status, ok := ks.ResourceStatuses[res.Key]
		if !ok || res.NeedUpdate(&status) {
			ops = append(ops, op.ResourceApplyOp(apiServer, res, !status.HasBeenSSA))
		}
	}
	return ops
}

func cleanOps(c *cke.Cluster, nf *NodeFilter) (ops []cke.Operator) {
	var apiServers, controllerManagers, schedulers, etcds, etcdRivers []*cke.Node

	for _, n := range c.Nodes {
		if !nf.status.NodeStatuses[n.Address].SSHConnected || n.ControlPlane {
			continue
		}

		st := nf.nodeStatus(n)
		if st.Etcd.Running && nf.EtcdIsGood() {
			etcds = append(etcds, n)
		}
		if st.APIServer.Running {
			apiServers = append(apiServers, n)
		}
		if st.ControllerManager.Running {
			controllerManagers = append(controllerManagers, n)
		}
		if st.Scheduler.Running {
			schedulers = append(schedulers, n)
		}
		if st.EtcdRivers.Running {
			etcdRivers = append(etcdRivers, n)
		}
	}

	if len(apiServers) > 0 {
		ops = append(ops, op.APIServerStopOp(apiServers))
	}
	if len(controllerManagers) > 0 {
		ops = append(ops, op.ControllerManagerStopOp(controllerManagers))
	}
	if len(schedulers) > 0 {
		ops = append(ops, op.SchedulerStopOp(schedulers))
	}
	if len(etcds) > 0 {
		ops = append(ops, op.EtcdStopOp(etcds))
	}
	if len(etcdRivers) > 0 {
		ops = append(ops, op.EtcdRiversStopOp(etcdRivers))
	}
	return ops
}
