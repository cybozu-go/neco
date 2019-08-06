package op

import (
	"path/filepath"
	"time"
)

const (
	// EtcdEndpointsName is the resource name for CKE-managed etcd
	EtcdEndpointsName = "cke-etcd"
	// EtcdServiceName is the resource name for CKE-managed etcd
	EtcdServiceName = EtcdEndpointsName

	etcdPKIPath = "/etc/etcd/pki"
	k8sPKIPath  = "/etc/kubernetes/pki"
)

const (
	// EtcdContainerName is container name of etcd
	EtcdContainerName = "etcd"
	// KubeAPIServerContainerName is name of kube-apiserver
	KubeAPIServerContainerName = "kube-apiserver"
	// KubeControllerManagerContainerName is name of kube-controller-manager
	KubeControllerManagerContainerName = "kube-controller-manager"
	// KubeProxyContainerName is container name of kube-proxy
	KubeProxyContainerName = "kube-proxy"
	// KubeSchedulerContainerName is container name of kube-scheduler
	KubeSchedulerContainerName = "kube-scheduler"
	// KubeletContainerName is container name of kubelet
	KubeletContainerName = "kubelet"
	// RiversContainerName is container name of rivers
	RiversContainerName = "rivers"
	// EtcdRiversContainerName is container name of etcd-rivers
	EtcdRiversContainerName = "etcd-rivers"

	// RiversUpstreamPort is upstream port of rivers container
	RiversUpstreamPort = 6443
	// RiversListenPort is listen port of rivers container
	RiversListenPort = 16443
	// EtcdRiversUpstreamPort is upstream port of etcd-rivers container
	EtcdRiversUpstreamPort = 2379
	// EtcdRiversListenPort is listen port of etcd-rivers container
	EtcdRiversListenPort = 12379

	// ClusterDNSAppName is app name of cluster DNS
	ClusterDNSAppName = "cluster-dns"
	// NodeDNSAppName is app name of node-dns
	NodeDNSAppName = "node-dns"

	// DefaultEtcdVolumeName is etcd default volume name
	DefaultEtcdVolumeName = "etcd-cke"

	// TimeoutDuration is default timeout duration
	TimeoutDuration = 5 * time.Second

	// CKELabelMaster is the label name added to control plane nodes
	CKELabelMaster = "cke.cybozu.com/master"
	// CKETaintMaster is the taint name added to control plane nodes
	CKETaintMaster = "cke.cybozu.com/master"

	// CKELabelAppName is application name
	CKELabelAppName = "cke.cybozu.com/appname"
	// EtcdBackupAppName is application name for etcdbackup
	EtcdBackupAppName = "etcdbackup"

	// PolicyConfigPath is a path for scheduler extender policy
	PolicyConfigPath = "/etc/kubernetes/scheduler/policy.cfg"
	// SchedulerConfigPath is a path for scheduler extender config
	SchedulerConfigPath = "/etc/kubernetes/scheduler/config.yml"
)

// EtcdPKIPath returns a certificate file path for k8s.
func EtcdPKIPath(p string) string {
	return filepath.Join(etcdPKIPath, p)
}

// K8sPKIPath returns a certificate file path for k8s.
func K8sPKIPath(p string) string {
	return filepath.Join(k8sPKIPath, p)
}
