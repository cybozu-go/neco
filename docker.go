package neco

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
	"github.com/mattn/go-isatty"
)

type dockerRuntime struct {
	dcTest bool
}

func newDockerRuntime(_ string) (ContainerRuntime, error) {
	_, err := os.Stat(DCTestFile)
	rt := dockerRuntime{err == nil}
	return rt, nil
}

func (rt dockerRuntime) ImageFullName(img ContainerImage) string {
	return img.FullName(!rt.dcTest)
}

func (rt dockerRuntime) Pull(ctx context.Context, img ContainerImage) error {
	cmd := well.CommandContext(ctx, "docker", "image", "ls", "--format", "{{ .Repository }}:{{ .Tag }}")
	data, err := cmd.Output()
	if err != nil {
		return err
	}

	fullname := rt.ImageFullName(img)
	for _, name := range strings.Fields(string(data)) {
		if name == fullname {
			return nil
		}
	}

	err = RetryWithSleep(ctx, 3, time.Second,
		func(ctx context.Context) error {
			return exec.CommandContext(ctx, "docker", "pull", "-q", fullname).Run()
		},
		func(err error) {
			log.Warn("docker: failed to pull a container image", map[string]interface{}{
				log.FnError: err,
				"image":     fullname,
			})
		},
	)
	if err == nil {
		log.Info("docker: pulled a container image", map[string]interface{}{
			"image": fullname,
		})
	}
	return err
}

func (rt dockerRuntime) Run(ctx context.Context, img ContainerImage, binds []Bind, args []string) error {
	runArgs := []string{"run", "--pull", "never", "--rm", "-u", "root:root", "--entrypoint="}
	for _, b := range binds {
		a := b.Source + ":" + b.Dest
		if b.ReadOnly {
			a += ":ro"
		}
		runArgs = append(runArgs, "-v", a)
	}
	runArgs = append(runArgs, rt.ImageFullName(img))
	runArgs = append(runArgs, args...)
	return well.CommandContext(ctx, "docker", runArgs...).Run()
}

func (rt dockerRuntime) Exec(ctx context.Context, name string, stdio bool, command []string) error {
	args := []string{"exec", "--privileged"}
	if stdio {
		if isatty.IsTerminal(os.Stdin.Fd()) {
			args = append(args, "-it")
		} else {
			args = append(args, "-i")
		}
	}
	args = append(args, name)
	args = append(args, command...)
	cmd := well.CommandContext(ctx, "docker", args...)
	if stdio {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	return cmd.Run()
}

func (rt dockerRuntime) IsRunning(img ContainerImage) (bool, error) {
	out, err := exec.Command("docker", "ps", "--format", "{{.Image}}").Output()
	if err != nil {
		return false, fmt.Errorf("failed to run docker ps: %w", err)
	}
	fullname := rt.ImageFullName(img)
	for _, name := range strings.Fields(string(out)) {
		if fullname == name {
			return true, nil
		}
	}
	return false, nil
}
