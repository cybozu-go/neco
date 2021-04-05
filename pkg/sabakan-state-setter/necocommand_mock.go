package sss

import (
	"context"
	"errors"
)

type necoCmdMockExecutor struct {
	powerState       map[string]string
	powerStopCount   map[string]int
	powerStatusCount map[string]int
	tpmClearCount    map[string]int
}

func newMockNecoCmdExecutor() *necoCmdMockExecutor {
	return &necoCmdMockExecutor{
		powerState:       map[string]string{},
		powerStopCount:   map[string]int{},
		powerStatusCount: map[string]int{},
		tpmClearCount:    map[string]int{},
	}
}

func (e *necoCmdMockExecutor) PowerStop(ctx context.Context, serial string) ([]byte, error) {
	e.powerStopCount[serial]++
	if e.powerState[serial] == "Off" {
		return []byte("log message"), errors.New("machine is already powered off")
	}
	e.powerState[serial] = "Off"
	return []byte("log message"), nil
}

func (e *necoCmdMockExecutor) PowerStatus(ctx context.Context, serial string) ([]byte, error) {
	e.powerStatusCount[serial]++
	if state, ok := e.powerState[serial]; ok {
		// When `neco power status` succeeds, only power status (e.g. "On", "Off") is output.
		// But the command fails, it outputs some error logs.
		return []byte(state + "\n"), nil
	}
	return []byte("log message"), errors.New("error")
}

func (e *necoCmdMockExecutor) TPMClear(ctx context.Context, serial string) ([]byte, error) {
	e.tpmClearCount[serial]++
	return []byte("log message"), nil
}

// test function
func (e *necoCmdMockExecutor) setPowerState(serial string, state string) {
	e.powerState[serial] = state
}

// test function
func (e *necoCmdMockExecutor) getPowerStopCount(serial string) int {
	return e.powerStopCount[serial]
}

// test function
func (e *necoCmdMockExecutor) getPowerStatusCount(serial string) int {
	return e.powerStatusCount[serial]
}

// test function
func (e *necoCmdMockExecutor) getTPMClearCount(serial string) int {
	return e.tpmClearCount[serial]
}
