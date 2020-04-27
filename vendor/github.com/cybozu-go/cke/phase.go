package cke

import "time"

// OperationPhase represents the processing status of CKE server.
type OperationPhase string

// Processing statuses of CKE server.
const (
	PhaseUpgrade         = OperationPhase("upgrade")
	PhaseRivers          = OperationPhase("rivers")
	PhaseEtcdBootAborted = OperationPhase("etcd-boot-aborted")
	PhaseEtcdBoot        = OperationPhase("etcd-boot")
	PhaseEtcdStart       = OperationPhase("etcd-start")
	PhaseEtcdWait        = OperationPhase("etcd-wait")
	PhaseK8sStart        = OperationPhase("k8s-start")
	PhaseEtcdMaintain    = OperationPhase("etcd-maintain")
	PhaseK8sMaintain     = OperationPhase("k8s-maintain")
	PhaseStopCP          = OperationPhase("stop-control-plane")
	PhaseCompleted       = OperationPhase("completed")
)

// AllOperationPhases contains all kinds of OperationPhases.
var AllOperationPhases = []OperationPhase{
	PhaseUpgrade,
	PhaseRivers,
	PhaseEtcdBootAborted,
	PhaseEtcdBoot,
	PhaseEtcdStart,
	PhaseEtcdWait,
	PhaseK8sStart,
	PhaseEtcdMaintain,
	PhaseK8sMaintain,
	PhaseStopCP,
	PhaseCompleted,
}

// ServerStatus represents the current server status.
type ServerStatus struct {
	Phase     OperationPhase `json:"phase"`
	Timestamp time.Time      `json:"timestamp"`
}
