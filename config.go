package neco

import "time"

// Default values
const (
	DefaultCheckUpdateInterval = 10 * time.Minute
	DefaultWorkerTimeout       = 60 * time.Minute
)

// Environements to use release or pre-release neco
const (
	StagingEnv = "staging"
	ProdEnv    = "prod"
)
