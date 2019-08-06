package cke

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"
)

const (
	ckeLabelName = "com.cybozu.cke"
)

// ContainerEngine defines interfaces for a container engine.
type ContainerEngine interface {
	// PullImage pulls an image.
	PullImage(img Image) error
	// Run runs a container as a foreground process.
	Run(img Image, binds []Mount, command string) error
	// RunWithInput runs a container as a foreground process with stdin as a string.
	RunWithInput(img Image, binds []Mount, command, input string) error
	// RunSystem runs the named container as a system service.
	RunSystem(name string, img Image, opts []string, params, extra ServiceParams) error
	// Exists returns if named system container exists.
	Exists(name string) (bool, error)
	// Stop stops the named system container.
	Stop(name string) error
	// Kill kills the named system container.
	Kill(name string) error
	// Remove removes the named system container.
	Remove(name string) error
	// Inspect returns ServiceStatus for the named container.
	Inspect(name []string) (map[string]ServiceStatus, error)
	// VolumeCreate creates a local volume.
	VolumeCreate(name string) error
	// VolumeRemove creates a local volume.
	VolumeRemove(name string) error
	// VolumeExists returns true if the named volume exists.
	VolumeExists(name string) (bool, error)
}

type ckeLabel struct {
	BuiltInParams ServiceParams `json:"builtin"`
	ExtraParams   ServiceParams `json:"extra"`
}

// Docker is an implementation of ContainerEngine.
func Docker(agent Agent) ContainerEngine {
	return docker{agent}
}

type docker struct {
	agent Agent
}

func (c docker) PullImage(img Image) error {
	stdout, stderr, err := c.agent.Run("docker image list --format '{{.Repository}}:{{.Tag}}'")
	if err != nil {
		return errors.Wrapf(err, "stdout: %s, stderr: %s", stdout, stderr)
	}

	for _, i := range strings.Split(string(stdout), "\n") {
		if img.Name() == i {
			return nil
		}
	}

	stdout, stderr, err = c.agent.Run("docker image pull " + img.Name())
	if err != nil {
		return errors.Wrapf(err, "stdout: %s, stderr: %s", stdout, stderr)
	}
	return nil
}

func (c docker) Run(img Image, binds []Mount, command string) error {
	args := []string{
		"docker",
		"run",
		"--log-driver=journald",
		"--rm",
		"--network=host",
		"--uts=host",
		"--read-only",
	}
	for _, m := range binds {
		o := "rw"
		if m.ReadOnly {
			o = "ro"
		}
		args = append(args, fmt.Sprintf("--volume=%s:%s:%s", m.Source, m.Destination, o))
	}
	args = append(args, img.Name(), command)

	_, _, err := c.agent.Run(strings.Join(args, " "))
	return err
}

func (c docker) RunWithInput(img Image, binds []Mount, command, input string) error {
	args := []string{
		"docker",
		"run",
		"--log-driver=journald",
		"--rm",
		"-i",
		"--network=host",
		"--uts=host",
		"--read-only",
	}
	for _, m := range binds {
		o := "rw"
		if m.ReadOnly {
			o = "ro"
		}
		args = append(args, fmt.Sprintf("--volume=%s:%s:%s", m.Source, m.Destination, o))
	}
	args = append(args, img.Name(), command)

	return c.agent.RunWithInput(strings.Join(args, " "), input)
}

func (c docker) RunSystem(name string, img Image, opts []string, params, extra ServiceParams) error {
	id, err := c.getID(name)
	if err != nil {
		return err
	}
	if len(id) != 0 {
		cmdline := "docker rm " + name
		stderr, stdout, err := c.agent.Run(cmdline)
		if err != nil {
			return errors.Wrapf(err, "cmdline: %s, stdout: %s, stderr: %s", cmdline, stdout, stderr)
		}
	}

	args := []string{
		"docker",
		"run",
		"--log-driver=journald",
		"-d",
		"--name=" + name,
		"--read-only",
		"--network=host",
		"--uts=host",
	}
	args = append(args, opts...)

	for _, m := range append(params.ExtraBinds, extra.ExtraBinds...) {
		var opts []string
		if m.ReadOnly {
			opts = append(opts, "ro")
		}
		if len(m.Propagation) > 0 {
			opts = append(opts, m.Propagation.String())
		}
		if len(m.Label) > 0 {
			opts = append(opts, m.Label.String())
		}
		args = append(args, fmt.Sprintf("--volume=%s:%s:%s", m.Source, m.Destination, strings.Join(opts, ",")))
	}
	for k, v := range params.ExtraEnvvar {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}
	for k, v := range extra.ExtraEnvvar {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	label := ckeLabel{
		BuiltInParams: params,
		ExtraParams:   extra,
	}
	data, err := json.Marshal(label)
	if err != nil {
		return err
	}
	labelFile, err := c.putData(ckeLabelName + "=" + string(data))
	if err != nil {
		return err
	}
	args = append(args, "--label-file="+labelFile)

	args = append(args, img.Name())

	args = append(args, params.ExtraArguments...)
	args = append(args, extra.ExtraArguments...)

	cmdline := strings.Join(args, " ")
	stdout, stderr, err := c.agent.Run(cmdline)
	if err != nil {
		return errors.Wrapf(err, "cmdline: %s, stdout: %s, stderr: %s", cmdline, stdout, stderr)
	}
	return nil
}

func (c docker) Stop(name string) error {
	cmdline := "docker container stop " + name
	stdout, stderr, err := c.agent.Run(cmdline)
	if err != nil {
		return errors.Wrapf(err, "cmdline: %s, stdout: %s, stderr: %s", cmdline, stdout, stderr)
	}
	return nil
}

func (c docker) Kill(name string) error {
	cmdline := "docker container kill " + name
	stdout, stderr, err := c.agent.Run(cmdline)
	if err != nil {
		return errors.Wrapf(err, "cmdline: %s, stdout: %s, stderr: %s", cmdline, stdout, stderr)
	}
	return nil
}

func (c docker) Remove(name string) error {
	cmdline := "docker container rm " + name
	stdout, stderr, err := c.agent.Run(cmdline)
	if err != nil {
		return errors.Wrapf(err, "cmdline: %s, stdout: %s, stderr: %s", cmdline, stdout, stderr)
	}
	return nil
}

func (c docker) putData(data string) (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	fileName := filepath.Join("/tmp", hex.EncodeToString(b))
	err = c.agent.RunWithInput("tee "+fileName, data)
	if err != nil {
		return "", err
	}
	return fileName, nil
}

func (c docker) getID(name string) (string, error) {
	cmdline := "docker ps -a --no-trunc --filter name=^/" + name + "$ --format {{.ID}}"
	stdout, stderr, err := c.agent.Run(cmdline)
	if err != nil {
		return "", errors.Wrapf(err, "cmdline: %s, stdout: %s, stderr: %s", cmdline, stdout, stderr)
	}
	return strings.TrimSpace(string(stdout)), nil
}

func (c docker) getIDs(names []string) (map[string]string, error) {
	filters := make([]string, len(names))
	for i, name := range names {
		filters[i] = "--filter name=^/" + name + "$"
	}
	cmdline := "docker ps -a --no-trunc " + strings.Join(filters, " ") + " --format {{.Names}}:{{.ID}}"
	stdout, stderr, err := c.agent.Run(cmdline)
	if err != nil {
		return nil, errors.Wrapf(err, "cmdline: %s, stdout: %s, stderr: %s", cmdline, stdout, stderr)
	}

	ids := make(map[string]string)
	scanner := bufio.NewScanner(bytes.NewReader(stdout))
	for scanner.Scan() {
		nameID := strings.Split(scanner.Text(), ":")
		ids[nameID[0]] = nameID[1]
	}
	return ids, nil
}

func (c docker) Exists(name string) (bool, error) {
	id, err := c.getID(name)
	if err != nil {
		return false, err
	}
	return len(id) != 0, nil
}

func (c docker) Inspect(names []string) (map[string]ServiceStatus, error) {
	retryCount := 0
RETRY:
	nameIds, err := c.getIDs(names)
	if err != nil {
		return nil, err
	}

	var ids []string
	for _, id := range nameIds {
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, nil
	}

	cmdline := "docker container inspect " + strings.Join(ids, " ")
	stdout, stderr, err := c.agent.Run(cmdline)
	if err != nil {
		retryCount++
		if retryCount >= 3 {
			return nil, errors.Wrapf(err, "cmdline: %s, stdout: %s, stderr: %s", cmdline, stdout, stderr)
		}
		goto RETRY
	}

	var djs []types.ContainerJSON
	err = json.Unmarshal(stdout, &djs)
	if err != nil {
		return nil, err
	}

	statuses := make(map[string]ServiceStatus)
	for _, dj := range djs {
		name := strings.TrimPrefix(dj.Name, "/")

		var params ckeLabel
		label := dj.Config.Labels[ckeLabelName]

		err = json.Unmarshal([]byte(label), &params)
		if err != nil {
			return nil, err
		}
		statuses[name] = ServiceStatus{
			Running:       dj.State.Running,
			Image:         dj.Config.Image,
			BuiltInParams: params.BuiltInParams,
			ExtraParams:   params.ExtraParams,
		}
	}

	return statuses, nil
}

func (c docker) VolumeCreate(name string) error {
	cmdline := "docker volume create " + name
	stdout, stderr, err := c.agent.Run(cmdline)
	if err != nil {
		return errors.Wrapf(err, "cmdline: %s, stdout: %s, stderr: %s", cmdline, stdout, stderr)
	}
	return nil
}

func (c docker) VolumeRemove(name string) error {
	cmdline := "docker volume remove " + name
	stdout, stderr, err := c.agent.Run(cmdline)
	if err != nil {
		return errors.Wrapf(err, "cmdline: %s, stdout: %s, stderr: %s", cmdline, stdout, stderr)
	}
	return nil
}

func (c docker) VolumeExists(name string) (bool, error) {
	cmdline := "docker volume list -q"
	stdout, stderr, err := c.agent.Run(cmdline)
	if err != nil {
		return false, errors.Wrapf(err, "cmdline: %s, stdout: %s, stderr: %s", cmdline, stdout, stderr)
	}

	for _, n := range strings.Split(string(stdout), "\n") {
		if n == name {
			return true, nil
		}
	}
	return false, nil
}
