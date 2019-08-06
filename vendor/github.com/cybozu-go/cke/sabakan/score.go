package sabakan

import (
	"time"
)

const (
	baseScore      = 10
	unhealthyScore = -3 * baseScore
)

func scoreByDays(days int) int {
	var score int
	if days > 250 {
		score += baseScore
	}
	if days > 500 {
		score += baseScore
	}
	if days > 1000 {
		score += baseScore
	}
	if days < -250 {
		score -= baseScore
	}
	if days < -500 {
		score -= baseScore
	}
	if days < -1000 {
		score -= baseScore
	}
	return score
}

func scoreMachine(m *Machine, rackCount map[int]int, ts time.Time) int {
	days := int(m.Spec.RetireDate.Sub(ts).Hours() / 24)
	score := scoreByDays(days)
	k := rackCount[m.Spec.Rack]
	if k > baseScore {
		k = baseScore
	}
	score -= k

	if m.Status.State != StateHealthy {
		score += unhealthyScore
	}

	return score
}
