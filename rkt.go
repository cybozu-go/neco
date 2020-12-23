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

	rktAuthFile = "/etc/rkt/auth.d/docker.json"
)

type rktRuntime struct {
	hasAuthFile bool
	env         []string
}

func newRktRuntime(proxy string) (ContainerRuntime, error) {
	rt := rktRuntime{}
	if proxy != "" {
		osenv := os.Environ()
		env := make([]string, 0, len(osenv)+2)
		env = append(env, osenv...)
		env = append(env, "https_proxy="+proxy, "http_proxy="+proxy)
		rt.env = env
	}

	_, err := os.Stat(rktAuthFile)
	switch {
	case err == nil:
		rt.hasAuthFile = true
	case os.IsNotExist(err):
	default:
		return nil, fmt.Errorf("failed to check %s: %w", rktAuthFile, err)
	}
	return rt, nil
}

func (rt rktRuntime) ImageFullName(img ContainerImage) string {
	return img.FullName(rt.hasAuthFile)
}

func (rt rktRuntime) Pull(ctx context.Context, img ContainerImage) error {
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
	fullname := rt.ImageFullName(img)
	for _, i := range list {
		if i.Name == fullname {
			return nil
		}
	}

	err = RetryWithSleep(ctx, retryCount, time.Second,
		func(ctx context.Context) error {
			cmd := exec.CommandContext(ctx, "rkt", "--insecure-options=image", "fetch", "--full", "docker://"+fullname)
			cmd.Env = rt.env
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

func (rt rktRuntime) Run(ctx context.Context, img ContainerImage, binds []Bind, args []string) error {
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

func (rt rktRuntime) Exec(ctx context.Context, name string, stdio bool, command []string) error {
	uuid, err := getRunningPodByApp(name)
	if err != nil {
		return err
	}

	rktArgs := []string{"enter", "--app", name, uuid}
	rktArgs = append(rktArgs, command...)
	cmd := well.CommandContext(ctx, "rkt", rktArgs...)
	if stdio {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	return cmd.Run()
}

func (rt rktRuntime) IsRunning(img ContainerImage) (bool, error) {
	uuid, err := getRunningPodByApp(img.Name)
	if err != nil {
		return false, err
	}
	data, err := exec.Command("rkt", "cat-manifest", uuid).Output()
	if err != nil {
		return false, fmt.Errorf("failed to run rkt cat-manifest %s: %w", uuid, err)
	}

	manifest := &struct {
		Apps []struct {
			Name  string `json:"name"`
			Image struct {
				Name   string `json:"name"`
				Labels []struct {
					Name  string `json:"name"`
					Value string `json:"value"`
				} `json:"labels"`
			} `json:"image"`
		} `json:"apps"`
	}{}
	if err := json.Unmarshal(data, manifest); err != nil {
		return false, fmt.Errorf("failed to parse rkt cat-manifest output: %s: %v", string(data), err)
	}

	fullName := rt.ImageFullName(img)
	for _, app := range manifest.Apps {
		if app.Name != img.Name {
			continue
		}
		for _, label := range app.Image.Labels {
			if label.Name != "version" {
				continue
			}
			return fmt.Sprintf("%s:%s", app.Image.Name, label.Value) == fullName, nil
		}
	}

	return false, fmt.Errorf("app %s is not found in rkt manifest: %s", img.Name, string(data))
}

var hasRktAuthFile bool

func init() {
	_, err := os.Stat(rktAuthFile)
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
	uuid, err := getRunningPodByApp(app)
	if err != nil {
		return nil, err
	}

	rktArgs := []string{"enter", "--app", app, uuid}
	rktArgs = append(rktArgs, args...)
	return well.CommandContext(ctx, "rkt", rktArgs...), nil
}

func getRunningPodByApp(app string) (string, error) {
	cmd := exec.Command("rkt", "list", "--format=json")
	data, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("rkt list failed: %w", err)
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
		return "", errors.New("failed to get rkt pod list")
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
	return "", errors.New("failed to find specified rkt app: " + app)
}
