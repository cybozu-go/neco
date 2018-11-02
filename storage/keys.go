package storage

import (
	"fmt"
	"strconv"
)

// etcd keys
const (
	KeyLeader            = "leader/"
	KeyBootserversPrefix = "bootservers/"
	KeyCurrent           = "current"
	KeyStatusPrefix      = "status/"
	KeyNotification      = "notification"
	KeyVaultUnsealKey    = "vault-unseal-key"
	KeyVaultRootToken    = "vault-root-token"
	KeyFinishPrefix      = "finish/"
	KeyContainersFormat  = "install/%d/containers/%s"
	KeyDebsFormat        = "install/%d/debs/%s"
)

func keyBootServer(lrn int) string {
	return KeyBootserversPrefix + strconv.Itoa(lrn)
}

func keyStatus(lrn int) string {
	return KeyStatusPrefix + strconv.Itoa(lrn)
}

func keyFinish(lrn int) string {
	return KeyFinishPrefix + strconv.Itoa(lrn)
}

func keyContainer(lrn int, name string) string {
	return fmt.Sprintf(KeyContainersFormat, lrn, name)
}

func keyDeb(lrn int, name string) string {
	return fmt.Sprintf(KeyDebsFormat, lrn, name)
}
