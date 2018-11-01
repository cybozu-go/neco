package neco

import "path/filepath"

// NecoDir is the base directory to store neco related files.
const NecoDir = "/etc/neco"

// EtcdDir is the base directory to store etcd related files.
const EtcdDir = "/etc/etcd"

// VaultDir is the base directory to store vault related files.
const VaultDir = "/etc/vault"

// Vault CA paths
const (
	CAServer     = "ca/server"
	CAEtcdPeer   = "ca/boot-etcd-peer"
	CAEtcdClient = "ca/boot-etcd-client"
)

// Vault TTL constants
const (
	TTL100Year = "876000h"
	TTL10Year  = "87600h"
)

// File locations
var (
	rackFile    = filepath.Join(NecoDir, "rack")
	clusterFile = filepath.Join(NecoDir, "cluster")

	ServerCAFile   = "/usr/local/share/ca-certificates/neco.crt"
	ServerCertFile = filepath.Join(NecoDir, "server.crt")
	ServerKeyFile  = filepath.Join(NecoDir, "server.key")

	EtcdPeerCAFile   = filepath.Join(EtcdDir, "ca-peer.crt")
	EtcdClientCAFile = filepath.Join(EtcdDir, "ca-client.crt")
	EtcdPeerCertFile = filepath.Join(EtcdDir, "peer.crt")
	EtcdPeerKeyFile  = filepath.Join(EtcdDir, "peer.key")
	EtcdConfFile     = filepath.Join(EtcdDir, "etcd.conf.yml")

	EtcdBackupCertFile = filepath.Join(EtcdDir, "backup.crt")
	EtcdBackupKeyFile  = filepath.Join(EtcdDir, "backup.key")

	VaultCertFile = filepath.Join(VaultDir, "etcd.crt")
	VaultKeyFile  = filepath.Join(VaultDir, "etcd.key")
	VaultConfFile = filepath.Join(VaultDir, "config.hcl")
)
