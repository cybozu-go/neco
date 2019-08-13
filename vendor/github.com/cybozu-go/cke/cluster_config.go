package cke

const (
	defaultEtcdVolumeName   = "etcd-cke"
	defaultClusterDomain    = "cluster.local"
	defaultEtcdBackupRotate = 14
)

// NewCluster creates Cluster
func NewCluster() *Cluster {
	return &Cluster{
		Options: Options{
			Etcd: EtcdParams{
				VolumeName: defaultEtcdVolumeName,
			},
			Kubelet: KubeletParams{
				Domain: defaultClusterDomain,
			},
		},
		EtcdBackup: EtcdBackup{
			Rotate: defaultEtcdBackupRotate,
		},
	}
}
