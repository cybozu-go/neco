package updater

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
)

type SlackServer struct {
	sv *httptest.Server

	ch    chan Payload
	queue []Payload
}

func NewSlackServer() *SlackServer {
	ch := make(chan Payload)

	s := &SlackServer{}
	s.sv = httptest.NewServer(s)
	s.ch = ch

	return s
}

func (s *SlackServer) WatchMessage() <-chan Payload {
	return s.ch
}

func (s *SlackServer) URL() string {
	return s.sv.URL
}

func (s *SlackServer) Close() {
	close(s.ch)
	s.sv.Close()
	s.sv = nil
}

func (s *SlackServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		w.WriteHeader(404)
		return
	}

	var payload Payload
	err := json.NewDecoder(req.Body).Decode(&payload)
	if err != nil {
		w.WriteHeader(400)
	}

	s.ch <- payload

	w.WriteHeader(200)
}
