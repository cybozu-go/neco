package storage

import (
	"fmt"
	"strconv"
)

// etcd keys
const (
	KeyLeader              = "leader/"
	KeyBootserversPrefix   = "bootservers/"
	KeyStatusPrefix        = "status/"
	KeyCurrent             = "status/current"
	KeyWorkerStatusPrefix  = "status/bootservers/"
	KeyNotificationSlack   = "config/notification/slack"
	KeyProxy               = "config/proxy"
	KeyEnv                 = "config/env"
	KeyCheckUpdateInterval = "config/check-update-interval"
	KeyWorkerTimeout       = "config/worker-timeout"
	KeyVaultUnsealKey      = "vault-unseal-key"
	KeyVaultRootToken      = "vault-root-token"
	KeyFinishPrefix        = "finish/"
	KeyContainersFormat    = "install/%d/containers/%s"
	KeyDebsFormat          = "install/%d/debs/%s"
)

func keyBootServer(lrn int) string {
	return KeyBootserversPrefix + strconv.Itoa(lrn)
}

func keyStatus(lrn int) string {
	return KeyWorkerStatusPrefix + strconv.Itoa(lrn)
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
