package app

import (
	"context"
	"net/http"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco/gcp"
	"golang.org/x/oauth2/google"
	compute "google.golang.org/api/compute/v1"
)

// Server is the API Server of GAE app
type Server struct {
	client *http.Client
	cfg    *gcp.Config
}

// NewServer creates a new Server
func NewServer(cfg *gcp.Config) (*Server, error) {
	client, err := google.DefaultClient(context.Background(), "https://www.googleapis.com/auth/compute")
	if err != nil {
		return nil, err
	}

	return &Server{
		client: client,
		cfg:    cfg,
	}, nil
}

// HandleShutdown handles REST API /shutdown
func (s Server) HandleShutdown(w http.ResponseWriter, r *http.Request) {
	gaeHeader := r.Header.Get("X-Appengine-Cron")
	if len(gaeHeader) == 0 {
		RenderError(r.Context(), w, APIErrForbidden)
		return
	}

	s.shutdown(w, r)
}

func (s Server) shutdown(w http.ResponseWriter, r *http.Request) {
	project := s.cfg.Common.Project
	zone := s.cfg.Common.Zone
	exclude := s.cfg.App.Shutdown.Exclude
	stop := s.cfg.App.Shutdown.Stop
	status := ShutdownStatus{}
	now := time.Now().UTC()
	expiration := s.cfg.App.Shutdown.Expiration

	service, err := compute.New(s.client)
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

		extendedAt, err := getExtendedAt(instance)
		if err != nil {
			RenderError(r.Context(), w, InternalServerError(err))
			return
		}
		elapsed := now.Sub(extendedAt)
		if elapsed.Seconds() < expiration.Seconds() {
			continue
		}

		if contain(instance.Name, stop) {
			_, err := service.Instances.Stop(project, zone, instance.Name).Do()
			if err != nil {
				RenderError(r.Context(), w, InternalServerError(err))
				continue
			}
			status.Stopped = append(status.Stopped, instance.Name)
		} else {
			_, err := service.Instances.Delete(project, zone, instance.Name).Do()
			if err != nil {
				RenderError(r.Context(), w, InternalServerError(err))
				continue
			}
			status.Deleted = append(status.Deleted, instance.Name)
		}
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

func getExtendedAt(instance *compute.Instance) (time.Time, error) {
	for _, metadata := range instance.Metadata.Items {
		if metadata.Key == gcp.MetadataKeyExtended {
			return time.Parse(time.RFC3339, *metadata.Value)
		}
	}
	return time.Parse(time.RFC3339, instance.CreationTimestamp)
}
