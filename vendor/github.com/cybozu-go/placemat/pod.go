package placemat

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
)

// PodInterfaceSpec represents a Pod's Interface definition in YAML
type PodInterfaceSpec struct {
	Network   string   `json:"network"`
	Addresses []string `json:"addresses,omitempty"`
}

// PodVolumeSpec represents a Pod's Volume definition in YAML
type PodVolumeSpec struct {
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Folder   string `json:"folder,omitempty"`
	ReadOnly bool   `json:"readonly"`
	Mode     string `json:"mode,omitempty"`
	UID      string `json:"uid,omitempty"`
	GID      string `json:"gid,omitempty"`
}

// PodAppMountSpec represents a App's Mount definition in YAML
type PodAppMountSpec struct {
	Volume string `json:"volume"`
	Target string `json:"target"`
}

// PodAppSpec represents a Pod's App definition in YAML
type PodAppSpec struct {
	Name           string            `json:"name"`
	Image          string            `json:"image"`
	ReadOnlyRootfs bool              `json:"readonly-rootfs"`
	User           string            `json:"user,omitempty"`
	Group          string            `json:"group,omitempty"`
	Exec           string            `json:"exec,omitempty"`
	Args           []string          `json:"args,omitempty"`
	Env            map[string]string `json:"env,omitempty"`
	CapsRetain     []string          `json:"caps-retain,omitempty"`
	Mount          []PodAppMountSpec `json:"mount,omitempty"`
}

// PodSpec represents a Pod specification in YAML
type PodSpec struct {
	Kind        string             `json:"kind"`
	Name        string             `json:"name"`
	InitScripts []string           `json:"init-scripts,omitempty"`
	Interfaces  []PodInterfaceSpec `json:"interfaces,omitempty"`
	Volumes     []*PodVolumeSpec   `json:"volumes,omitempty"`
	Apps        []*PodAppSpec      `json:"apps"`
}

// PodVolume is an interface of a volume for Pod.
type PodVolume interface {
	// Name returns the volume name.
	Name() string
	// Resolve resolves references in the volume definition.
	Resolve(*Cluster) error
	// Spec returns a command-line argument for the volume.
	Spec() string
}

// NewPodVolume makes a PodVolume, or returns an error.
func NewPodVolume(spec *PodVolumeSpec) (PodVolume, error) {
	if len(spec.Name) == 0 {
		return nil, errors.New("invalid pod volume name")
	}
	switch spec.Kind {
	case "host":
		return newHostPodVolume(spec.Name, spec.Folder, spec.ReadOnly), nil
	case "empty":
		return newEmptyPodVolume(spec.Name, spec.Mode, spec.UID, spec.GID), nil
	}

	return nil, errors.New("invalid kind of pod volume: " + spec.Kind)
}

type hostPodVolume struct {
	name       string
	folderName string
	folder     *DataFolder
	readOnly   bool
}

func (v *hostPodVolume) Name() string {
	return v.name
}

func (v *hostPodVolume) Resolve(c *Cluster) error {
	df, err := c.GetDataFolder(v.folderName)
	if err != nil {
		return err
	}
	v.folder = df
	return nil
}

func (v *hostPodVolume) Spec() string {
	return fmt.Sprintf("%s,kind=host,source=%s,readOnly=%v", v.name, v.folder.Path(), v.readOnly)
}

func newHostPodVolume(name, folder string, readOnly bool) PodVolume {
	return &hostPodVolume{name, folder, nil, readOnly}
}

type emptyPodVolume struct {
	name string
	mode string
	uid  string
	gid  string
}

func (v *emptyPodVolume) Name() string {
	return v.name
}

func (v *emptyPodVolume) Resolve(c *Cluster) error {
	return nil
}

func (v *emptyPodVolume) Spec() string {
	buf := make([]byte, 0, 32)
	buf = append(buf, v.name...)
	buf = append(buf, ",kind=empty,readOnly=false"...)
	if len(v.mode) > 0 {
		buf = append(buf, ",mode="...)
		buf = append(buf, v.mode...)
	}
	if len(v.uid) > 0 {
		buf = append(buf, ",uid="...)
		buf = append(buf, v.uid...)
	}
	if len(v.gid) > 0 {
		buf = append(buf, ",gid="...)
		buf = append(buf, v.gid...)
	}
	return string(buf)
}

func newEmptyPodVolume(name, mode, uid, gid string) PodVolume {
	return &emptyPodVolume{name, mode, uid, gid}
}

func (a *PodAppSpec) appendParams(params []string, podname string) []string {
	params = append(params, []string{
		a.Image,
		"--name", a.Name,
		"--user-label", "name=" + podname,
	}...)
	if a.ReadOnlyRootfs {
		params = append(params, "--readonly-rootfs=true")
	}
	if len(a.User) > 0 {
		params = append(params, "--user="+a.User)
	}
	if len(a.Group) > 0 {
		params = append(params, "--group="+a.Group)
	}
	if len(a.Exec) > 0 {
		params = append(params, []string{"--exec", a.Exec}...)
	}
	for k, v := range a.Env {
		params = append(params, fmt.Sprintf("--set-env=%s=%s", k, v))
	}
	if len(a.CapsRetain) > 0 {
		params = append(params, "--caps-retain="+strings.Join(a.CapsRetain, ","))
	}
	for _, mp := range a.Mount {
		t := fmt.Sprintf("volume=%s,target=%s", mp.Volume, mp.Target)
		params = append(params, []string{"--mount", t}...)
	}
	if len(a.Args) > 0 {
		params = append(params, "--")
		params = append(params, a.Args...)
	}

	return params
}

// Pod represents a pod resource.
type Pod struct {
	*PodSpec
	initScripts []string
	volumes     []PodVolume
	networks    []*Network
	veths       map[string]string
	uuid        string
	pid         int
}

// NewPod creates a Pod from spec.
func NewPod(spec *PodSpec) (*Pod, error) {
	p := &Pod{
		PodSpec: spec,
		veths:   make(map[string]string),
	}

	if len(spec.Name) == 0 {
		return nil, errors.New("pod name is empty")
	}

	for _, script := range spec.InitScripts {
		script, err := filepath.Abs(script)
		if err != nil {
			return nil, err
		}
		_, err = os.Stat(script)
		if err != nil {
			return nil, err
		}
		p.initScripts = append(p.initScripts, script)
	}

	for _, vs := range spec.Volumes {
		vol, err := NewPodVolume(vs)
		if err != nil {
			return nil, err
		}
		p.volumes = append(p.volumes, vol)
	}

	if len(spec.Apps) == 0 {
		return nil, errors.New("no app for pod " + spec.Name)
	}

	return p, nil
}

// CleanupPods cleans file created at runtime for rkt.
func CleanupPods(r *Runtime, pods []*Pod) {
	for _, pod := range pods {
		uuidFile := filepath.Join(r.runDir, pod.Name+".uuid")
		_, err := os.Stat(uuidFile)
		if err == nil {
			err = os.Remove(uuidFile)
			if err != nil {
				log.Warn("failed to clean file", map[string]interface{}{
					"filename": uuidFile,
				})
			}
		}
	}
}

// Resolve resolves references to other resources in the cluster.
func (p *Pod) Resolve(c *Cluster) error {
	for _, iface := range p.Interfaces {
		network, err := c.GetNetwork(iface.Network)
		if err != nil {
			return err
		}
		p.networks = append(p.networks, network)
	}

	for _, v := range p.volumes {
		err := v.Resolve(c)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Pod) appendParams(params []string) []string {
	params = append(params, []string{"--hostname", p.Name}...)
	for _, v := range p.volumes {
		params = append(params, []string{"--volume", v.Spec()}...)
	}

	addDDD := false
	for _, a := range p.Apps {
		if addDDD {
			params = append(params, "---")
		}
		params = a.appendParams(params, p.Name)
		addDDD = len(a.Args) > 0
	}
	return params
}

func fetchImage(ctx context.Context, image string) error {
	log.Info("fetching image", map[string]interface{}{
		"image": image,
	})
	args := []string{
		"--pull-policy=new",
		"--insecure-options=image",
		"fetch",
		image,
	}
	return well.CommandContext(ctx, "rkt", args...).Run()
}

// Prepare fetches container images to run Pod.
func (p *Pod) Prepare(ctx context.Context) error {
	for _, a := range p.Apps {
		err := fetchImage(ctx, a.Image)
		if err != nil {
			return err
		}
	}
	return nil
}

func makePodNS(ctx context.Context, pod string, veths []string, ips map[string][]string) error {
	log.Info("Creating Pod network namespace", map[string]interface{}{"pod": pod})
	ns := "pm_" + pod
	cmds := [][]string{
		{"ip", "netns", "add", ns},
		{"ip", "netns", "exec", ns, "ip", "link", "set", "lo", "up"},
		// enable IP forwarding
		{"ip", "netns", "exec", ns, "sysctl", "-w", v4ForwardKey + "=1"},
		{"ip", "netns", "exec", ns, "sysctl", "-w", v6ForwardKey + "=1"},
		// 127.0.0.1 is auto-assigned to lo.
		//{"ip", "netns", "exec", ns, "ip", "a", "add", "127.0.0.1/8", "dev", "lo"},
	}
	for i, veth := range veths {
		eth := fmt.Sprintf("eth%d", i)
		cmds = append(cmds, []string{
			"ip", "link", "set", veth, "netns", ns, "name", eth, "up",
		})
		for _, ip := range ips[veth] {
			cmds = append(cmds, []string{
				"ip", "netns", "exec", ns, "ip", "a", "add", ip, "dev", eth,
			})
		}
	}
	return execCommands(ctx, cmds)
}

func runInPodNS(ctx context.Context, pod string, script string) error {
	return well.CommandContext(ctx, "ip", "netns", "exec", "pm_"+pod, script).Run()
}

func deletePodNS(ctx context.Context, pod string) error {
	return well.CommandContext(ctx, "ip", "netns", "del", "pm_"+pod).Run()
}

// Start starts the Pod using rkt.  It does not return until
// the process finishes or ctx is cancelled.
func (p *Pod) Start(ctx context.Context, r *Runtime) error {
	veths := make([]string, len(p.networks))
	ips := make(map[string][]string)
	for i, n := range p.networks {
		veth, vethInNS, err := n.CreateVeth()
		if err != nil {
			return err
		}
		veths[i] = vethInNS
		ips[vethInNS] = p.Interfaces[i].Addresses
		p.veths[n.Name] = veth
	}

	err := makePodNS(ctx, p.Name, veths, ips)
	if err != nil {
		return err
	}
	defer deletePodNS(context.Background(), p.Name)

	for _, script := range p.initScripts {
		err := runInPodNS(ctx, p.Name, script)
		if err != nil {
			return err
		}
	}

	uuidFile := filepath.Join(r.runDir, p.Name+".uuid")
	if _, err = os.Stat(uuidFile); err == nil {
		return errors.New("uuid file is already exist: " + uuidFile)
	}
	defer os.RemoveAll(uuidFile)

	params := []string{
		"--insecure-options=all-run",
		"run",
		"--net=host",
		"--dns=host",
		"--uuid-file-save=" + uuidFile,
	}
	params = p.appendParams(params)

	log.Info("rkt run", map[string]interface{}{"name": p.Name, "params": params})
	args := []string{
		"netns", "exec", "pm_" + p.Name, "rkt",
	}
	args = append(args, params...)
	rkt := exec.Command("ip", args...)
	rkt.Stdout = newColoredLogWriter("rkt", p.Name, os.Stdout)
	rkt.Stderr = newColoredLogWriter("rkt", p.Name, os.Stderr)
	err = rkt.Start()
	if err != nil {
		log.Error("failed to start rkt", map[string]interface{}{
			log.FnError: err,
		})
		return err
	}

	count := 60
RETRY:
	count--
	if count == 0 {
		return errors.New("could not read a uuid file of the container: " + p.Name)
	}
	u, err := ioutil.ReadFile(uuidFile)
	if err != nil {
		time.Sleep(time.Second)
		goto RETRY
	}
	uuid := strings.TrimSpace(string(u))
	if len(uuid) != 36 || rkt.Process.Pid == 0 {
		time.Sleep(time.Second)
		goto RETRY
	}
	p.uuid = uuid
	p.pid = rkt.Process.Pid

	go func() {
		<-ctx.Done()
		log.Info("kill pod", map[string]interface{}{
			"pod":  p.Name,
			"pid":  p.pid,
			"uuid": p.uuid,
		})
		rkt.Process.Signal(syscall.SIGTERM)
	}()
	err = rkt.Wait()
	log.Info("pod finished", map[string]interface{}{
		"pod":  p.Name,
		"pid":  p.pid,
		"uuid": p.uuid,
	})
	return err
}
