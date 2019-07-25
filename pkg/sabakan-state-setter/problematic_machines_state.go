package main

import (
	"encoding/json"
	"io"
	"os"
)

type problematicMachine struct {
	Name           string `json:"name"`
	Serial         string `json:"serial"`
	State          string `json:"state"`
	FirstDetection string `json:"first_detection"`
}

func parseProblematicMachinesFile(f io.Reader) ([]problematicMachine, error) {
	var res []problematicMachine
	err := json.NewDecoder(f).Decode(res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func writeProblematicMachines(filename string, pms []problematicMachine) error {
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	err = json.NewEncoder(f).Encode(pms)
	if err != nil {
		return err
	}
	return f.Sync()
}

func isProblematicState(target string) bool {
	for _, s := range problematicStates {
		if s == target {
			return true
		}
	}
	return false
}
