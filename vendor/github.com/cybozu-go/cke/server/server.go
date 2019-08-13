package server

import (
	"context"
	"net/http"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/etcdserver/api/v3rpc/rpctypes"
	"github.com/cybozu-go/cke"
)

// Server is the cke server.
type Server struct {
	EtcdClient *clientv3.Client
	Timeout    time.Duration
}

type version struct {
	Version string `json:"version"`
}

type health struct {
	Health string `json:"health"`
}

func (s Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet && r.URL.Path == "/version" {
		s.handleVersion(w, r)
	} else if r.Method == http.MethodGet && r.URL.Path == "/health" {
		s.handleHealth(w, r)
	} else {
		renderError(r.Context(), w, APIErrNotFound)
	}
}

func (s Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	renderJSON(w, version{
		Version: cke.Version,
	}, http.StatusOK)
}

func (s Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctxWithTimeout, cancel := context.WithTimeout(r.Context(), s.Timeout)
	defer cancel()

	_, err := s.EtcdClient.Get(ctxWithTimeout, "health")
	if err == nil || err == rpctypes.ErrPermissionDenied {
		renderJSON(w, health{
			Health: "healthy",
		}, http.StatusOK)
	} else {
		renderJSON(w, health{
			Health: "unhealthy",
		}, http.StatusInternalServerError)
	}
}
