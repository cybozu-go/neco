package main

type problematicMachine struct {
	Name           string `json:"name"`
	Serial         string `json:"serial"`
	State          string `json:"state"`
	FirstDetection string `json:"first_detection"`
}

func isProblematicState(target string) bool {
	for _, s := range problematicStates {
		if s == target {
			return true
		}
	}
	return false
}
