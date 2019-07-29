package sss

var problematicStates = []string{"unreachable", "unhealthy"}

func isProblematicState(target string) bool {
	for _, s := range problematicStates {
		if s == target {
			return true
		}
	}
	return false
}
