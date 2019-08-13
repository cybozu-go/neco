package cke

import "errors"

// Constraints is a set of conditions that a cluster must satisfy
type Constraints struct {
	ControlPlaneCount int `json:"control-plane-count"`
	MinimumWorkers    int `json:"minimum-workers"`
	MaximumWorkers    int `json:"maximum-workers"`
}

// Check checks the cluster satisfies the constraints
func (c *Constraints) Check(cluster *Cluster) error {
	cpCount := 0
	nodeCount := len(cluster.Nodes)

	for _, n := range cluster.Nodes {
		if n.ControlPlane {
			cpCount++
		}
	}

	if cpCount != c.ControlPlaneCount {
		return errors.New("number of control planes is not equal to the constraint")
	}
	workerCount := nodeCount - cpCount
	if c.MaximumWorkers != 0 && workerCount > c.MaximumWorkers {
		return errors.New("number of worker nodes exceeds the maximum")
	}
	if workerCount < c.MinimumWorkers {
		return errors.New("number of worker nodes is less than the minimum")
	}
	return nil
}

// DefaultConstraints returns the default constraints
func DefaultConstraints() *Constraints {
	return &Constraints{
		ControlPlaneCount: 1,
		MinimumWorkers:    1,
		MaximumWorkers:    0,
	}
}
