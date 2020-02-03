package app

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco/gcp"
	"github.com/nlopes/slack"
	"golang.org/x/oauth2/google"
	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
)

// Server is the API Server of GAE app
type Server struct {
	client *http.Client
	cfg    *gcp.Config
}

var errShutdownMetadataNotFound = errors.New(gcp.MetadataKeyShutdownAt + " is not found")

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

	service, err := compute.NewService(r.Context(), option.WithHTTPClient(s.client))
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

		shutdownAt, err := getShutdownAt(instance)
		if err != nil {
			if err != errShutdownMetadataNotFound {
				RenderError(r.Context(), w, InternalServerError(err))
				return
			}
		}
		if err != errShutdownMetadataNotFound {
			if now.Sub(shutdownAt) >= 0 {
				_, err := service.Instances.Delete(project, zone, instance.Name).Do()
				if err != nil {
					RenderError(r.Context(), w, InternalServerError(err))
					continue
				}
				status.Deleted = append(status.Deleted, instance.Name)
			}
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

func getShutdownAt(instance *compute.Instance) (time.Time, error) {
	for _, metadata := range instance.Metadata.Items {
		if metadata.Key == gcp.MetadataKeyShutdownAt {
			return time.Parse(time.RFC3339, *metadata.Value)
		}
	}
	return time.Time{}, errShutdownMetadataNotFound
}

// HandleHandle handles REST API /extend
func (s Server) HandleExtend(w http.ResponseWriter, r *http.Request) {
	s.extend(w, r)
}

func (s Server) extend(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	bodyRaw, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error("failed to read body", map[string]interface{}{
			log.FnError: err,
		})
		RenderError(r.Context(), w, InternalServerError(err))
		return
	}

	body, _ := url.QueryUnescape(string(bodyRaw))
	body = strings.Replace(body, "payload=", "", 1)

	project := s.cfg.Common.Project
	zone := s.cfg.Common.Zone

	service, err := compute.NewService(r.Context(), option.WithHTTPClient(s.client))
	if err != nil {
		log.Error("failed to create client", map[string]interface{}{
			log.FnError: err,
		})
		RenderError(r.Context(), w, InternalServerError(err))
		return
	}

	p, err := service.Projects.Get(project).Do()
	if err != nil {
		log.Error("failed to get project", map[string]interface{}{
			"project": project,
		})
		RenderError(r.Context(), w, InternalServerError(err))
		return
	}
	verificationToken := ""
	for _, item := range p.CommonInstanceMetadata.Items {
		if item.Key == "SLACK_VERIFICATION_TOKEN" {
			verificationToken = *item.Value
		}
	}
	if len(verificationToken) == 0 {
		log.Error("token not found", map[string]interface{}{})
		RenderError(r.Context(), w, InternalServerError(errors.New("SLACK_VERIFICATION_TOKEN not found")))
		return
	}

	var message slack.InteractionCallback
	err = json.Unmarshal([]byte(body), &message)
	if err != nil {
		log.Error("failed to unmarshal body", map[string]interface{}{
			log.FnError: err,
		})
		RenderError(r.Context(), w, InternalServerError(err))
		return
	}

	if message.Token != verificationToken {
		log.Error("invalid token", map[string]interface{}{})
		RenderError(r.Context(), w, InternalServerError(errors.New("invalid token")))
		return
	}

	instance := message.ActionCallback.BlockActions[0].Value
	target, err := service.Instances.Get(project, zone, instance).Do()
	if err != nil {
		log.Error("failed to get target instance", map[string]interface{}{
			log.FnError: err,
			"project":   project,
			"zone":      zone,
			"instance":  instance,
		})
		RenderError(r.Context(), w, InternalServerError(err))
		return
	}

	shutdownAt := time.Now().UTC().Add(2 * time.Hour).Format(time.RFC3339)
	found := false
	metadata := target.Metadata
	for _, m := range metadata.Items {
		if m.Key == gcp.MetadataKeyShutdownAt {
			m.Value = &shutdownAt
			found = true
			break
		}
	}
	if !found {
		metadata.Items = append(metadata.Items, &compute.MetadataItems{
			Key:   gcp.MetadataKeyShutdownAt,
			Value: &shutdownAt,
		})
	}

	_, err = service.Instances.SetMetadata(project, zone, instance, metadata).Do()
	if err != nil {
		log.Error("failed to set metadata", map[string]interface{}{
			log.FnError:   err,
			"project":     project,
			"zone":        zone,
			"instance":    instance,
			"shutdown_at": shutdownAt,
		})
		RenderError(r.Context(), w, InternalServerError(err))
		return
	}

	log.Info("extended instance", map[string]interface{}{
		"project":     project,
		"zone":        zone,
		"instance":    instance,
		"shutdown_at": shutdownAt,
	})

	RenderJSON(w, ExtendStatus{Extended: instance}, http.StatusOK)
}
