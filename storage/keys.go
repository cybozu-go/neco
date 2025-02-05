package storage

import (
	"fmt"
	"strconv"
)

// etcd keys
const (
	KeySabakanStateSetterLeader    = "leader/sabakan-state-setter/"
	KeyUpdaterLeader               = "leader/updater/"
	KeyWorkerLeader                = "leader/worker/"
	KeyNecoRebooterLeader          = "leader/neco-rebooter/"
	KeyInfoPrefix                  = "info/"
	KeyBootserversPrefix           = "info/bootservers/"
	KeyNecoRelease                 = "info/neco-release"
	KeySSHPubkey                   = "info/ssh-pubkey"
	KeyStatusPrefix                = "status/"
	KeyCurrent                     = "status/current"
	KeyWorkerStatusPrefix          = "status/bootservers/"
	KeyContentsPrefix              = "contents/"
	KeySabakanContents             = "contents/sabakan"
	KeyCKEContents                 = "contents/cke"
	KeyDHCPJSONContents            = "contents/dhcp.json"
	KeyCKETemplateContents         = "contents/cke-template"
	KeyUserResourcesContents       = "contents/user-resources"
	KeyConfigPrefix                = "config/"
	KeyNotificationSlack           = "config/notification/slack"
	KeyProxy                       = "config/proxy"
	KeyEnv                         = "config/env"
	KeyCheckUpdateInterval         = "config/check-update-interval"
	KeyWorkerTimeout               = "config/worker-timeout"
	KeyGitHubToken                 = "config/github-token"
	KeyNodeProxy                   = "config/node-proxy"
	KeyExternalIPAddressBlock      = "config/external-ip-address-block"
	KeyLBAddressBlockDefault       = "config/lb-address-block-default"
	KeyLBAddressBlockBastion       = "config/lb-address-block-bastion"
	KeyLBAddressBlockInternet      = "config/lb-address-block-internet"
	KeyLBAddressBlockInternetCN    = "config/lb-address-block-internet-cn"
	KeyReleaseTime                 = "config/release-time"
	KeyReleaseTimeZone             = "config/release-timezone"
	KeyVaultUnsealKey              = "vault-unseal-key"
	KeyVaultRootToken              = "vault-root-token"
	KeyFinishPrefix                = "finish/"
	KeyContainersFormat            = "install/%d/containers/%s"
	KeyDebsFormat                  = "install/%d/debs/%s"
	KeyInstallPrefix               = "install/"
	KeyBMCBMCUser                  = "bmc/bmc-user"
	KeyBMCIPMIUser                 = "bmc/ipmi-user"
	KeyBMCIPMIPassword             = "bmc/ipmi-password"
	KeyBMCRepairUser               = "bmc/repair-user"
	KeyBMCRepairPassword           = "bmc/repair-password"
	KeyTeleportAuthToken           = "teleport/auth-token"
	KeyCKEWeight                   = "cke/weight"
	KeyNecoRebooterRebootList      = "neco-rebooter/reboot-list/"
	KeyNecoRebooterWriteIndex      = "neco-rebooter/write-index"
	KeyNecoRebooterProcessingGroup = "neco-rebooter/processing-group"
	KeyNecoRebooterIsEnabled       = "neco-rebooter/is-enabled"
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
