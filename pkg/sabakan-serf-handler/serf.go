package main

import (
	"errors"
	"strings"
)

// Event types stored in SERF_EVENT envvar
const (
	EventMemberJoin   = "member-join"
	EventMemberLeave  = "member-leave"
	EventMemberFailed = "member-failed"
)

// Payload represents payload in serf event
type Payload struct {
	Name    string
	Address string
	Role    string
	Tags    map[string]string
}

// ParsePayload parses an input line of serf event
func ParsePayload(line string) (*Payload, error) {
	fields := strings.Split(line, "\t")
	if len(fields) != 4 {
		return nil, errors.New("line does not have 4 fields")
	}

	kvs := strings.Split(fields[3], ",")
	tags := make(map[string]string)
	for _, kv := range kvs {
		pair := strings.Split(kv, "=")
		if len(pair) == 1 {
			tags[pair[0]] = ""
		} else {
			tags[pair[0]] = pair[1]
		}
	}

	ev := &Payload{
		Name:    fields[0],
		Address: fields[1],
		Role:    fields[2],
		Tags:    tags,
	}
	return ev, nil
}
