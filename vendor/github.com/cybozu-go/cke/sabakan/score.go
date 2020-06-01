package sabakan

import (
	"time"
)

const (
	// maxCountPerRack should be more than max machine num per rack + 1
	maxCountPerRack = 100
	// healthyScore is addted when the machine status is healthy.
	healthyScore = 1000
)

func scoreByDays(days int) int {
	var score int
	if days > 250 {
		score++
	}
	if days > 500 {
		score++
	}
	if days > 1000 {
		score++
	}
	if days < -250 {
		score--
	}
	if days < -500 {
		score--
	}
	if days < -1000 {
		score--
	}
	return score
}

func scoreMachine(m *Machine, rackCount map[int]int, ts time.Time) int {
	rackScore := maxCountPerRack - rackCount[m.Spec.Rack]

	days := int(m.Spec.RetireDate.Sub(ts).Hours() / 24)
	daysScore := scoreByDays(days)

	return rackScore*10 + daysScore
}

func scoreMachineWithHealthStatus(m *Machine, rackCount map[int]int, ts time.Time) int {
	score := scoreMachine(m, rackCount, ts)
	if m.Status.State != StateHealthy {
		return score
	}
	return healthyScore + score
}

func filterHealthyMachinesByRole(ms []*Machine, role string) []*Machine {
	var filtered []*Machine
	for _, m := range ms {
		if m.Status.State != StateHealthy {
			continue
		}
		if role != "" && m.Spec.Role != role {
			continue
		}
		filtered = append(filtered, m)
	}

	return filtered
}
