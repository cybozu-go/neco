package storage

import (
	"fmt"
	"strconv"
)

// etcd keys
const (
	KeyUpdaterLeader         = "leader/updater/"
	KeyWorkerLeader          = "leader/worker/"
	KeyInfoPrefix            = "info/"
	KeyBootserversPrefix     = "info/bootservers/"
	KeyNecoRelease           = "info/neco-release"
	KeySSHPubkey             = "info/ssh-pubkey"
	KeyStatusPrefix          = "status/"
	KeyCurrent               = "status/current"
	KeyWorkerStatusPrefix    = "status/bootservers/"
	KeySabakanContents       = "contents/sabakan"
	KeyCKEContents           = "contents/cke"
	KeyDHCPJSONContents      = "contents/dhcp.json"
	KeyCKETemplateContents   = "contents/cke-template"
	KeyUserResourcesContents = "contents/user-resources"
	KeyConfigPrefix          = "config/"
	KeyNotificationSlack     = "config/notification/slack"
	KeyProxy                 = "config/proxy"
	KeyQuayUsername          = "config/quay-username"
	KeyQuayPassword          = "config/quay-password"
	KeyEnv                   = "config/env"
	KeyCheckUpdateInterval   = "config/check-update-interval"
	KeyWorkerTimeout         = "config/worker-timeout"
	KeyGitHubToken           = "config/github-token"
	KeyVaultUnsealKey        = "vault-unseal-key"
	KeyVaultRootToken        = "vault-root-token"
	KeyFinishPrefix          = "finish/"
	KeyContainersFormat      = "install/%d/containers/%s"
	KeyDebsFormat            = "install/%d/debs/%s"
	KeyInstallPrefix         = "install/"
	KeyBMCBMCUser            = "bmc/bmc-user"
	KeyBMCIPMIUser           = "bmc/ipmi-user"
	KeyBMCIPMIPassword       = "bmc/ipmi-password"
)

func keyBootServer(lrn int) string {
	return KeyBootserversPrefix + strconv.Itoa(lrn)
}

func keyInstall(lrn int) string {
	return KeyInstallPrefix + strconv.Itoa(lrn)
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
