package main

import (
	"encoding/json"
	"io"
)

type problematicMachine struct {
	Name           string `json:"name"`
	Serial         string `json:"serial"`
	State          string `json:"state"`
	FirstDetection string `json:"first_detection"`
}

func parseProblematicMachinesFile(f io.Reader) ([]problematicMachine, error) {
	res := []problematicMachine{}
	err := json.NewDecoder(f).Decode(res)
	if err != nil {
		return nil, err
	}
	return res, nil
}
