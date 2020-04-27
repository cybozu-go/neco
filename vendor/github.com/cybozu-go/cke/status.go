package cke

import (
	"github.com/coreos/etcd/etcdserver/etcdserverpb"
	schedulerv1 "k8s.io/kube-scheduler/config/v1"

	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

// EtcdClusterStatus is the status of the etcd cluster.
type EtcdClusterStatus struct {
	IsHealthy     bool
	Members       map[string]*etcdserverpb.Member
	InSyncMembers map[string]bool
}

// ClusterDNSStatus contains cluster resolver status.
type ClusterDNSStatus struct {
	ConfigMap *corev1.ConfigMap
	ClusterIP string
}

// NodeDNSStatus contains node local resolver status.
type NodeDNSStatus struct {
	ConfigMap *corev1.ConfigMap
}

// EtcdBackupStatus is the status of etcdbackup
type EtcdBackupStatus struct {
	ConfigMap *corev1.ConfigMap
	CronJob   *batchv1beta1.CronJob
	Pod       *corev1.Pod
	Secret    *corev1.Secret
	Service   *corev1.Service
}

// KubernetesClusterStatus contains kubernetes cluster configurations
type KubernetesClusterStatus struct {
	IsControlPlaneReady bool
	Nodes               []corev1.Node
	DNSService          *corev1.Service
	ClusterDNS          ClusterDNSStatus
	NodeDNS             NodeDNSStatus
	MasterEndpoints     *corev1.Endpoints
	EtcdService         *corev1.Service
	EtcdEndpoints       *corev1.Endpoints
	EtcdBackup          EtcdBackupStatus
	ResourceStatuses    map[string]ResourceStatus
}

// ResourceStatus represents the status of registered K8s resources
type ResourceStatus struct {
	// Annotations is the copy of metadata.annotations
	Annotations map[string]string
	// HasBeenSSA indicates that this resource has been already updated by server-side apply
	HasBeenSSA bool
}

// IsReady returns the cluster condition whether or not Pod can be scheduled
func (s KubernetesClusterStatus) IsReady(cluster *Cluster) bool {
	if !s.IsControlPlaneReady {
		return false
	}
	clusterNodesSize := len(cluster.Nodes)
	if clusterNodesSize == 0 {
		return false
	}
	currentReady := 0
	for _, n := range s.Nodes {
		for _, cond := range n.Status.Conditions {
			if cond.Type != corev1.NodeReady {
				continue
			}
			if cond.Status == corev1.ConditionTrue {
				currentReady++
				break
			}
		}
	}
	return clusterNodesSize/2 < currentReady
}

// SetResourceStatus sets status of the resource.
func (s KubernetesClusterStatus) SetResourceStatus(rkey string, ann map[string]string, isManaged bool) {
	s.ResourceStatuses[rkey] = ResourceStatus{Annotations: ann, HasBeenSSA: isManaged}
}

// ClusterStatus represents the working cluster status.
// The structure reflects Cluster, of course.
type ClusterStatus struct {
	ConfigVersion string
	Name          string
	NodeStatuses  map[string]*NodeStatus // keys are IP address strings.

	Etcd       EtcdClusterStatus
	Kubernetes KubernetesClusterStatus
}

// NodeStatus status of a node.
type NodeStatus struct {
	SSHConnected      bool
	Etcd              EtcdStatus
	Rivers            ServiceStatus
	EtcdRivers        ServiceStatus
	APIServer         KubeComponentStatus
	ControllerManager KubeComponentStatus
	Scheduler         SchedulerStatus
	Proxy             KubeComponentStatus
	Kubelet           KubeletStatus
	Labels            map[string]string // are labels for k8s Node resource.
}

// ServiceStatus represents statuses of a service.
//
// If Running is false, the service is not running on the node.
// ExtraXX are extra parameters of the running service, if any.
type ServiceStatus struct {
	Running       bool
	Image         string
	BuiltInParams ServiceParams
	ExtraParams   ServiceParams
}

// EtcdStatus is the status of kubelet.
type EtcdStatus struct {
	ServiceStatus
	HasData bool
}

// KubeComponentStatus represents service status and endpoint's health
type KubeComponentStatus struct {
	ServiceStatus
	IsHealthy bool
}

// SchedulerStatus represents kube-scheduler status and health
type SchedulerStatus struct {
	ServiceStatus
	IsHealthy bool
	Extenders []schedulerv1.Extender
}

// KubeletStatus represents kubelet status and health
type KubeletStatus struct {
	ServiceStatus
	IsHealthy                   bool
	Domain                      string
	AllowSwap                   bool
	ContainerLogMaxSize         string
	ContainerLogMaxFiles        int32
	NeedUpdateBlockPVsUpToV1_16 []string
}
