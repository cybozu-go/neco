package main

import (
	"reflect"
	"testing"
)

func TestParsePayload(t *testing.T) {
	line := "mitchellh.local	127.0.0.1	web	role=web,datacenter=east"
	ev, err := ParsePayload(line)
	if err != nil {
		t.Fatal(err)
	}

	expected := &Event{
		Name:    "mitchellh.local",
		Address: "127.0.0.1",
		Role:    "web",
		Tags:    map[string]string{"role": "web", "datacenter": "east"},
	}
	if !reflect.DeepEqual(ev, expected) {
		t.Errorf("!reflect.DeepEqual(ev, expected): %#v", ev)
	}
}
