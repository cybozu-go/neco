package app

import (
	"net/http"

	"github.com/cybozu-go/neco/gcp"
)

// Server is the API Server of GAE app
type Server struct {
	cfg *gcp.Config
}

// NewServer creates a new Server
func NewServer(cfg *gcp.Config) *Server {
	return &Server{
		cfg: cfg,
	}
}

func (s Server) HandleShutdown(w http.ResponseWriter, r *http.Request) {
	RenderJSON(w, ShutdownStatus{
		Stopped: []string{"aaa"},
		Deleted: []string{"bbb"},
	}, http.StatusOK)
}

func (s Server) HandleExtend(w http.ResponseWriter, r *http.Request) {
	RenderJSON(w, ExtendStatus{
		Extended:       []string{"aaa"},
		Time:           111,
		AvailableUntil: "today",
	}, http.StatusOK)
}
