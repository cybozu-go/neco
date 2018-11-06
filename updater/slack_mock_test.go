package updater

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
)

type SlackServer struct {
	sv *httptest.Server

	ch    chan<- Payload
	done  chan<- struct{}
	queue []Payload
}

func NewSlackServer() *SlackServer {
	ch := make(chan Payload)
	done := make(chan struct{})

	s := &SlackServer{}
	s.sv = httptest.NewServer(s)
	s.ch = ch
	s.done = done

	go func() {
		for {
			select {
			case p := <-ch:
				s.queue = append(s.queue, p)
			case <-done:
				return
			}
		}

	}()

	return s
}

func (s *SlackServer) Messages() []Payload {
	return s.queue
}

func (s *SlackServer) URL() string {
	return s.sv.URL
}

func (s *SlackServer) Close() {
	s.done <- struct{}{}
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
