package neco

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
)

const retryCount = 3

// RktImage represents rkt image information
type RktImage struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// FetchContainer fetches a container image
func FetchContainer(ctx context.Context, name string, env []string) error {
	img, err := CurrentArtifacts.FindContainerImage(name)
	if err != nil {
		return err
	}
	fullname := img.FullName()

	cmd := well.CommandContext(ctx, "rkt", "image", "list", "--format=json")
	data, err := cmd.Output()
	if err != nil {
		return err
	}
	var list []RktImage
	err = json.Unmarshal(data, &list)
	if err != nil {
		return err
	}
	for _, i := range list {
		if i.Name == fullname {
			return nil
		}
	}

	for i := 0; i < retryCount; i++ {
		fetchCmd := exec.CommandContext(ctx, "rkt", "--insecure-options=image", "fetch", "--full", "docker://"+fullname)
		fetchCmd.Env = env
		err = fetchCmd.Run()
		if err == nil {
			log.Info("rkt: fetched a container image", map[string]interface{}{
				"image": fullname,
			})
			return nil
		}
		log.Warn("rkt: failed to fetch a container image", map[string]interface{}{
			log.FnError: err,
			"image":     fullname,
		})
	}
	return err
}

// Bind represents a host bind mount rule.
type Bind struct {
	Name     string
	Source   string
	Dest     string
	ReadOnly bool
}

// Args returns command-line arguments for rkt.
func (b Bind) Args() []string {
	return []string{
		fmt.Sprintf("--volume=%s,kind=host,source=%s,readOnly=%v", b.Name, b.Source, b.ReadOnly),
		fmt.Sprintf("--mount=volume=%s,target=%s", b.Name, b.Dest),
	}
}

// RunContainer runs container in front.
func RunContainer(ctx context.Context, name string, binds []Bind, args []string) error {
	img, err := CurrentArtifacts.FindContainerImage(name)
	if err != nil {
		return err
	}

	rktArgs := []string{"run", "--pull-policy=never"}
	for _, b := range binds {
		rktArgs = append(rktArgs, b.Args()...)
	}
	rktArgs = append(rktArgs, img.FullName())
	rktArgs = append(rktArgs, args...)

	cmd := well.CommandContext(ctx, "rkt", rktArgs...)
	return cmd.Run()
}

// RktPod represents rkt pod information
type RktPod struct {
	Name     string   `json:"name"`
	State    string   `json:"state"`
	AppNames []string `json:"app_names"`
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
	var pods []RktPod
	for i := 0; i < retryCount; i++ {
		cmd := well.CommandContext(ctx, "rkt", "list", "--format=json")
		data, err := cmd.Output()
		if err != nil {
			return "", err
		}

		err = json.Unmarshal(data, &pods)
		if err != nil {
			return "", err
		}

		// for unknown reason, "rkt list" sometimes returns "null"
		if len(pods) != 0 {
			break
		}
	}

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
