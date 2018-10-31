package storage

import "strconv"

const (
	KeyLeader            = "leader/"
	KeyBootserversPrefix = "bootservers/"
	KeyCurrent           = "current"
	KeyStatusPrefix      = "status/"
	KeyNotification      = "notification"
	KeyVaultUnsealKey    = "vault-unseal-key"
	KeyVaultRootToken    = "vault-root-token"
)

func keyBootServers(lrn int) string {
	return KeyBootserversPrefix + strconv.Itoa(lrn)
}

func keyStatus(lrn int) string {
	return KeyStatusPrefix + strconv.Itoa(lrn)
}
