package main

import (
	"bufio"
	"errors"
	"io"
	"os"

	"github.com/cybozu-go/log"
)

var sabakanEndpoint = "http://127.0.0.1:10080"

func run(event string, stdin io.Reader) error {
	switch event {
	case EventMemberJoin, EventMemberLeave, EventMemberFailed:
	default:
		return errors.New("unsupported event: " + event)
	}

	c := sabakan{endpoint: sabakanEndpoint}

	s := bufio.NewScanner(stdin)
	for s.Scan() {
		line := s.Text()
		ev, err := ParsePayload(line)
		if err != nil {
			log.Warn("event parse error", map[string]interface{}{
				log.FnError: err,
				"event":     event,
				"payload":   line,
			})
			continue
		}

		serial := ev.Tags["serial"]
		os := ev.Tags["os-version"]
		if len(serial) == 0 {
			log.Warn("empty serial number", map[string]interface{}{
				"event":   event,
				"payload": line,
			})
			continue
		}

		switch event {
		case EventMemberJoin:
			err = c.updateState(serial, StateHealthy)
			if err != nil {
				break
			}
			err = c.updateOSVersion(serial, os)
		case EventMemberLeave:
			err = c.updateState(serial, StateUninitialized)
		case EventMemberFailed:
			err = c.updateState(serial, StateUnreachable)
		}
		if err != nil {
			log.Warn("failed to update sabakan status", map[string]interface{}{
				"event":     event,
				"payload":   line,
				log.FnError: err,
			})
			continue
		}

	}
	return nil
}

func main() {
	e := os.Getenv("SERF_EVENT")
	if len(e) == 0 {
		log.ErrorExit(errors.New("SERF_EVENT does not set"))
	}
	err := run(e, os.Stdin)
	if err != nil {
		log.ErrorExit(err)
	}
}
