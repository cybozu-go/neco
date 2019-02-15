package app

import (
	"net/http"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco/gcp"
	compute "google.golang.org/api/compute/v1"
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

// HandleShutdown handles REST API /shutdown
func (s Server) HandleShutdown(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Common.Project == "neco-test" {
		s.shutdownForNecoTest(w, r)
		return
	}
	s.shutdown(w, r)
}

func (s Server) shutdownForNecoTest(w http.ResponseWriter, r *http.Request) {
	return
}

func (s Server) shutdown(w http.ResponseWriter, r *http.Request) {
	project := s.cfg.Common.Project
	zone := s.cfg.Common.Zone
	exclude := s.cfg.App.Shutdown.Exclude
	delete := s.cfg.App.Shutdown.Delete
	status := ShutdownStatus{}

	service, err := compute.New(&http.Client{})
	if err != nil {
		RenderError(r.Context(), w, InternalServerError(err))
		return
	}
	instanceList, err := service.Instances.List(project, zone).Do()
	if err != nil {
		RenderError(r.Context(), w, InternalServerError(err))
		return
	}

	for _, instance := range instanceList.Items {
		if contain(instance.Name, exclude) {
			continue
		}

		if contain(instance.Name, delete) {
			_, err := service.Instances.Delete(project, zone, instance.Name).Do()
			if err != nil {
				RenderError(r.Context(), w, InternalServerError(err))
				continue
			}
			status.Deleted = append(status.Deleted, instance.Name)
		}

		_, err := service.Instances.Stop(project, zone, instance.Name).Do()
		if err != nil {
			RenderError(r.Context(), w, InternalServerError(err))
			continue
		}
		status.Stopped = append(status.Stopped, instance.Name)
	}

	log.Info("shutdown instances", map[string]interface{}{
		"deleted": status.Deleted,
		"stopped": status.Stopped,
	})
	RenderJSON(w, status, http.StatusOK)
}

func contain(name string, items []string) bool {
	for _, item := range items {
		if name == item {
			return true
		}
	}
	return false
}
