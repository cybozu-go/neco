package main

import (
	"errors"
	"strings"
)

const (
	EventMemberJoin   = "member-join"
	EventMemberLeave  = "member-leave"
	EventMemberFailed = "member-failed"
)

type Event struct {
	Name    string
	Address string
	Role    string
	Tags    map[string]string
}

func ParsePayload(line string) (*Event, error) {
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

	ev := &Event{
		Name:    fields[0],
		Address: fields[1],
		Role:    fields[2],
		Tags:    tags,
	}
	return ev, nil
}
