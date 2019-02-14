package app

import (
	"net/http"
	"strings"

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

// Handler implements http.Handler
func (s Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/shutdown") {
		s.handleShutdown(w, r)
		return
	} else if strings.HasPrefix(r.URL.Path, "/extend") {
		s.handleExtend(w, r)
		return
	}
	RenderError(r.Context(), w, APIErrBadRequest)
}

func (s Server) handleShutdown(w http.ResponseWriter, r *http.Request) {
	RenderJSON(w, ShutdownStatus{
		Stopped: []string{"aaa"},
		Deleted: []string{"bbb"},
	}, http.StatusOK)
}

func (s Server) handleExtend(w http.ResponseWriter, r *http.Request) {
	RenderJSON(w, ExtendStatus{
		Extended:       []string{"aaa"},
		Time:           111,
		AvailableUntil: "today",
	}, http.StatusOK)
}
