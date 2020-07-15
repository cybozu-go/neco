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
	commonZone := s.cfg.Common.Zone
	addZones := s.cfg.App.Shutdown.AdditionalZones
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

	targetZones := append([]string{commonZone}, addZones...)
	var errList []error
	for _, zone := range targetZones {
		instanceList, err := service.Instances.List(project, zone).Do()
		if err != nil {
			errList = append(errList, err)
			continue
		}

		for _, instance := range instanceList.Items {
			if contain(instance.Name, exclude) {
				continue
			}

			shutdownAt, err := getShutdownAt(instance)
			switch err {
			case errShutdownMetadataNotFound:
			case nil:
				if now.Sub(shutdownAt) >= 0 {
					_, err := service.Instances.Delete(project, zone, instance.Name).Do()
					if err != nil {
						errList = append(errList, err)
						continue
					}
					status.Deleted = append(status.Deleted, instance.Name)
				}
				continue
			default:
				errList = append(errList, err)
				continue
			}

			creationTime, err := time.Parse(time.RFC3339, instance.CreationTimestamp)
			if err != nil {
				errList = append(errList, err)
				continue
			}
			elapsed := now.Sub(creationTime)
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
	}

	log.Info("shutdown instances", map[string]interface{}{
		"deleted": status.Deleted,
		"stopped": status.Stopped,
	})
	if len(errList) != 0 {
		log.Error("shutdown failed", map[string]interface{}{
			"errors": errList,
		})
		RenderError(r.Context(), w, InternalServerError(errList[0]))
		return
	}
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

func getShutdownAt(instance *compute.Instance) (time.Time, error) {
	for _, metadata := range instance.Metadata.Items {
		if metadata.Key == gcp.MetadataKeyShutdownAt {
			return time.Parse(time.RFC3339, *metadata.Value)
		}
	}
	return time.Time{}, errShutdownMetadataNotFound
}

// HandleExtend handles REST API /extend
func (s Server) HandleExtend(w http.ResponseWriter, r *http.Request) {
	s.extend(w, r)
}

func (s Server) findGCPInstanceByName(service *compute.Service, project string, instance string) (*compute.Instance, string, error) {
	commonZone := s.cfg.Common.Zone
	addZones := s.cfg.App.Shutdown.AdditionalZones
	targetZones := append([]string{commonZone}, addZones...)

	var err error
	for _, zone := range targetZones {
		var target *compute.Instance
		target, err = service.Instances.Get(project, zone, instance).Do()
		if err == nil {
			return target, zone, nil
		}
	}
	log.Error("failed to get target instance", map[string]interface{}{
		log.FnError: err,
		"project":   project,
		"zones":     targetZones,
		"instance":  instance,
	})
	return nil, "", err
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

	body, err := url.QueryUnescape(string(bodyRaw))
	if err != nil {
		log.Error("failed to unescape query", map[string]interface{}{
			log.FnError: err,
		})
		RenderError(r.Context(), w, InternalServerError(err))
		return
	}
	body = strings.Replace(body, "payload=", "", 1)

	project := s.cfg.Common.Project

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

	if len(message.ActionCallback.BlockActions) < 1 {
		log.Error("block_actions is empty", map[string]interface{}{})
		RenderError(r.Context(), w, InternalServerError(errors.New("block_actions is empty")))
		return
	}
	instance := message.ActionCallback.BlockActions[0].Value

	// Find GCP instance from all target zones
	target, zone, err := s.findGCPInstanceByName(service, project, instance)
	if err != nil {
		RenderError(r.Context(), w, InternalServerError(err))
		return
	}

	// Extend instance lifetime
	shutdownTime, err := gcp.ConvertLocalTimeToUTC(s.cfg.App.Shutdown.Timezone, s.cfg.App.Shutdown.ShutdownAt)
	if err != nil {
		RenderError(r.Context(), w, InternalServerError(err))
		return
	}
	if shutdownTime.Before(time.Now()) {
		shutdownTime = shutdownTime.AddDate(0, 0, 1)
	}
	shutdownAt := shutdownTime.Format(time.RFC3339)
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
