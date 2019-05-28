package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func newServer() *server {
	s := &server{
		os:    make(map[string]string),
		state: make(map[string]string),
	}
	s.server = httptest.NewServer(http.HandlerFunc(s.handle))
	return s
}

type server struct {
	os    map[string]string
	state map[string]string

	server *httptest.Server
}

func (s server) handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		w.WriteHeader(400)
		return
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(500)
		return
	}
	r.Body.Close()

	if strings.HasPrefix(r.URL.Path, UpdateStatePrefix) {
		serial := r.URL.Path[len(UpdateStatePrefix):]
		s.state[serial] = string(body)
	} else if strings.HasPrefix(r.URL.Path, UpdateLabelPrefix) {
		serial := r.URL.Path[len(UpdateLabelPrefix):]
		var data map[string]string
		err := json.Unmarshal(body, &data)
		if err != nil {
			w.WriteHeader(500)
		}
		s.os[serial] = data["os-version"]
	} else {
		w.WriteHeader(404)
	}
	w.WriteHeader(200)
}

func (s server) Close() {
	s.server.Close()
}

func testRunMemberJoin(t *testing.T) {
	s := newServer()
	defer s.Close()

	sabakanEndpoint = s.server.URL

	err := run(EventMemberJoin, strings.NewReader(
		`mitchellh.local	127.0.0.1	web	os-version=18.04,serial=xxxx1000
mitchellh.local	127.0.0.1	web	os-version=16.04,serial=xxxx1001
`))
	if err != nil {
		t.Fatal(err)
	}

	os := map[string]string{"xxxx1000": "18.04", "xxxx1001": "16.04"}
	if !reflect.DeepEqual(s.os, os) {
		t.Fatal("!reflect.DeepEqual(s.os, os)", s.os)
	}
	state := map[string]string{"xxxx1000": StateHealthy, "xxxx1001": StateHealthy}
	if !reflect.DeepEqual(s.state, state) {
		t.Fatal("!reflect.DeepEqual(s.state, state)", s.state)
	}
}
func testRunMemberLeave(t *testing.T) {
	s := newServer()
	defer s.Close()

	sabakanEndpoint = s.server.URL

	err := run(EventMemberLeave, strings.NewReader(
		`mitchellh.local	127.0.0.1	web	os-version=18.04,serial=xxxx1000
mitchellh.local	127.0.0.1	web	os-version=16.04,serial=xxxx1001
`))
	if err != nil {
		t.Fatal(err)
	}

	state := map[string]string{"xxxx1000": StateUninitialized, "xxxx1001": StateUninitialized}
	if !reflect.DeepEqual(s.state, state) {
		t.Fatal("!reflect.DeepEqual(s.state, state)", s.state)
	}
}
func testRunMemberFailed(t *testing.T) {
	s := newServer()
	defer s.Close()

	sabakanEndpoint = s.server.URL

	err := run(EventMemberFailed, strings.NewReader(
		`mitchellh.local	127.0.0.1	web	os-version=18.04,serial=xxxx1000
mitchellh.local	127.0.0.1	web	os-version=16.04,serial=xxxx1001
`))
	if err != nil {
		t.Fatal(err)
	}

	state := map[string]string{"xxxx1000": StateUnreachable, "xxxx1001": StateUnreachable}
	if !reflect.DeepEqual(s.state, state) {
		t.Fatal("!reflect.DeepEqual(s.state, state)", s.state)
	}
}

func TestRun(t *testing.T) {
	t.Run("MemberJoin", testRunMemberJoin)
	t.Run("MemberLeave", testRunMemberLeave)
	t.Run("MemberFailed", testRunMemberFailed)
}
