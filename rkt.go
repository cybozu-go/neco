package neco

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
)

const (
	retryCount = 3

	dockerAuthFile = "/etc/rkt/auth.d/docker.json"
)

var hasRktAuthFile bool

func init() {
	_, err := os.Stat(dockerAuthFile)
	switch {
	case err == nil:
		hasRktAuthFile = true
	case os.IsNotExist(err):
	default:
		panic(err)
	}
}

// ContainerFullName returns full container's name for the name
func ContainerFullName(name string) (string, error) {
	img, err := CurrentArtifacts.FindContainerImage(name)
	if err != nil {
		return "", err
	}
	return img.FullName(hasRktAuthFile), nil
}

// FetchContainer fetches a container image
func FetchContainer(ctx context.Context, fullname string, env []string) error {
	cmd := well.CommandContext(ctx, "rkt", "image", "list", "--format=json")
	data, err := cmd.Output()
	if err != nil {
		return err
	}

	type rktImage struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	var list []rktImage
	err = json.Unmarshal(data, &list)
	if err != nil {
		return err
	}
	for _, i := range list {
		if i.Name == fullname {
			return nil
		}
	}

	err = RetryWithSleep(ctx, retryCount, time.Second,
		func(ctx context.Context) error {
			cmd := exec.CommandContext(ctx, "rkt", "--insecure-options=image", "fetch", "--full", "docker://"+fullname)
			cmd.Env = env
			return cmd.Run()
		},
		func(err error) {
			log.Warn("rkt: failed to fetch a container image", map[string]interface{}{
				log.FnError: err,
				"image":     fullname,
			})
		},
	)
	if err == nil {
		log.Info("rkt: fetched a container image", map[string]interface{}{
			"image": fullname,
		})
	}
	return err
}

// HTTPProxyEnv returns os.Environ() with http_proxy/https_proxy if proxy is not empty
func HTTPProxyEnv(proxy string) []string {
	osenv := os.Environ()
	env := make([]string, len(osenv))
	copy(env, osenv)
	if proxy != "" {
		env = append(env, "https_proxy="+proxy, "http_proxy="+proxy)
	}
	return env
}

// Bind represents a host bind mount rule.
type Bind struct {
	Name     string
	Source   string
	Dest     string
	ReadOnly bool
}

// RunContainer runs container in front.
func RunContainer(ctx context.Context, name string, binds []Bind, args []string) error {
	img, err := CurrentArtifacts.FindContainerImage(name)
	if err != nil {
		return err
	}

	rktArgs := []string{"run", "--pull-policy=never"}
	for _, b := range binds {
		rktArgs = append(rktArgs,
			fmt.Sprintf("--volume=%s,kind=host,source=%s,readOnly=%v", b.Name, b.Source, b.ReadOnly),
			fmt.Sprintf("--mount=volume=%s,target=%s", b.Name, b.Dest),
		)
	}
	rktArgs = append(rktArgs, img.FullName(hasRktAuthFile))
	rktArgs = append(rktArgs, args...)

	cmd := well.CommandContext(ctx, "rkt", rktArgs...)
	return cmd.Run()
}

// EnterContainerAppCommand returns well.LogCmd to enter the named app.
func EnterContainerAppCommand(ctx context.Context, app string, args []string) (*well.LogCmd, error) {
	uuid, err := getRunningPodByApp(ctx, app)
	if err != nil {
		return nil, err
	}

	rktArgs := []string{"enter", "--app", app, uuid}
	rktArgs = append(rktArgs, args...)
	return well.CommandContext(ctx, "rkt", rktArgs...), nil
}

func getRunningPodByApp(ctx context.Context, app string) (string, error) {
	cmd := well.CommandContext(ctx, "rkt", "list", "--format=json")
	data, err := cmd.Output()
	if err != nil {
		return "", err
	}

	type rktPod struct {
		Name     string   `json:"name"`
		State    string   `json:"state"`
		AppNames []string `json:"app_names"`
	}

	var pods []rktPod
	err = json.Unmarshal(data, &pods)
	if err != nil {
		return "", err
	}

	// for unknown reason, "rkt list" sometimes returns "null"
	if len(pods) == 0 {
		return "", errors.New("failed to get pod list")
	}

	for _, pod := range pods {
		if pod.State != "running" {
			continue
		}
		for _, appName := range pod.AppNames {
			if appName == app {
				return pod.Name, nil
			}
		}
	}
	return "", errors.New("failed to find specified app")
}
