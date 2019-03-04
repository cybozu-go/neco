package placemat

import (
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"net"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
)

const (
	defaultOVMFCodePath  = "/usr/share/OVMF/OVMF_CODE.fd"
	defaultOVMFVarsPath  = "/usr/share/OVMF/OVMF_VARS.fd"
	defaultRebootTimeout = 30 * time.Second
)

// SMBIOSConfig represents a Node's SMBIOS definition in YAML
type SMBIOSConfig struct {
	Manufacturer string `yaml:"manufacturer,omitempty"`
	Product      string `yaml:"product,omitempty"`
	Serial       string `yaml:"serial,omitempty"`
}

// NodeSpec represents a Node specification in YAML
type NodeSpec struct {
	Kind         string           `yaml:"kind"`
	Name         string           `yaml:"name"`
	Interfaces   []string         `yaml:"interfaces,omitempty"`
	Volumes      []NodeVolumeSpec `yaml:"volumes,omitempty"`
	IgnitionFile string           `yaml:"ignition,omitempty"`
	CPU          int              `yaml:"cpu,omitempty"`
	Memory       string           `yaml:"memory,omitempty"`
	UEFI         bool             `yaml:"uefi,omitempty"`
	SMBIOS       SMBIOSConfig     `yaml:"smbios,omitempty"`
}

// Node represents a virtual machine.
type Node struct {
	*NodeSpec
	networks []*Network
	taps     map[string]string
	volumes  []NodeVolume
}

func createNodeVolume(spec NodeVolumeSpec) (NodeVolume, error) {
	switch spec.Kind {
	case "image":
		if spec.Image == "" {
			return nil, errors.New("image volume must specify an image name")
		}
		return NewImageVolume(spec.Name, spec.Image, spec.CopyOnWrite), nil
	case "localds":
		if spec.UserData == "" {
			return nil, errors.New("localds volume must specify user-data")
		}
		return NewLocalDSVolume(spec.Name, spec.UserData, spec.NetworkConfig), nil
	case "raw":
		if spec.Size == "" {
			return nil, errors.New("raw volume must specify size")
		}
		return NewRawVolume(spec.Name, spec.Size), nil
	case "vvfat":
		if spec.Folder == "" {
			return nil, errors.New("VVFAT volume must specify a folder name")
		}
		return NewVVFATVolume(spec.Name, spec.Folder), nil
	default:
		return nil, errors.New("unknown volume kind: " + spec.Kind)
	}
}

// NewNode creates a Node from spec.
func NewNode(spec *NodeSpec) (*Node, error) {
	n := &Node{
		NodeSpec: spec,
		taps:     make(map[string]string),
	}
	if spec.Name == "" {
		return nil, errors.New("node name is empty")
	}

	for _, v := range spec.Volumes {
		vol, err := createNodeVolume(v)
		if err != nil {
			return nil, err
		}
		n.volumes = append(n.volumes, vol)
	}
	return n, nil
}

// Resolve resolves references to other resources in the cluster.
func (n *Node) Resolve(c *Cluster) error {
	for _, iface := range n.Interfaces {
		network, err := c.GetNetwork(iface)
		if err != nil {
			return err
		}
		n.networks = append(n.networks, network)
	}

	for _, vol := range n.volumes {
		err := vol.Resolve(c)
		if err != nil {
			return err
		}
	}

	return nil
}

// CleanupNodes cleans files created at runtime for QEMU.
func CleanupNodes(r *Runtime, nodes []*Node) {
	for _, d := range nodes {
		files := []string{
			r.guestSocketPath(d.Name),
			r.monitorSocketPath(d.Name),
			r.socketPath(d.Name),
		}
		for _, f := range files {
			_, err := os.Stat(f)
			if err == nil {
				err = os.Remove(f)
				if err != nil {
					log.Warn("failed to clean file", map[string]interface{}{
						"filename": f,
					})
				}
			}
		}
	}
}

func nodeSerial(name string) string {
	return fmt.Sprintf("%x", sha1.Sum([]byte(name)))
}

func (n *Node) qemuParams(r *Runtime) []string {
	params := []string{"-enable-kvm"}

	if n.IgnitionFile != "" {
		params = append(params, "-fw_cfg")
		params = append(params, "opt/com.coreos/config,file="+n.IgnitionFile)
	}

	if n.CPU != 0 {
		params = append(params, "-smp", strconv.Itoa(n.CPU))
	}
	if n.Memory != "" {
		params = append(params, "-m", n.Memory)
	}
	if !r.graphic {
		p := r.socketPath(n.Name)
		params = append(params, "-nographic")
		params = append(params, "-serial", "unix:"+p+",server,nowait")
	}
	if n.UEFI {
		p := r.nvramPath(n.Name)
		params = append(params, "-drive", "if=pflash,file="+defaultOVMFCodePath+",format=raw,readonly")
		params = append(params, "-drive", "if=pflash,file="+p+",format=raw")
	}

	smbios := "type=1"
	if n.SMBIOS.Manufacturer != "" {
		smbios += ",manufacturer=" + n.SMBIOS.Manufacturer
	}
	if n.SMBIOS.Product != "" {
		smbios += ",product=" + n.SMBIOS.Product
	}
	if n.SMBIOS.Serial == "" {
		n.SMBIOS.Serial = nodeSerial(n.Name)
	}
	smbios += ",serial=" + n.SMBIOS.Serial
	params = append(params, "-smbios", smbios)
	return params
}

// Start starts the Node as a QEMU process.
// This will not wait the process termination; instead, it returns the process information.
func (n *Node) Start(ctx context.Context, r *Runtime, nodeCh chan<- bmcInfo) (*NodeVM, error) {
	params := n.qemuParams(r)

	for _, vol := range n.volumes {
		vname := vol.Name()
		log.Info("Creating volume", map[string]interface{}{"node": n.Name, "volume": vname})
		p := filepath.Join(r.dataDir, "volumes", n.Name)
		err := os.MkdirAll(p, 0755)
		if err != nil {
			return nil, err
		}
		args, err := vol.Create(ctx, p)
		if err != nil {
			return nil, err
		}

		params = append(params, args...)
	}

	for _, br := range n.networks {
		tap, err := br.CreateTap()
		if err != nil {
			return nil, err
		}
		n.taps[br.Name] = tap

		netdev := "tap,id=" + br.Name + ",ifname=" + tap + ",script=no,downscript=no"
		if vhostNetSupported {
			netdev += ",vhost=on"
		}

		params = append(params, "-netdev", netdev)

		devParams := []string{
			"virtio-net-pci",
			fmt.Sprintf("netdev=%s", br.Name),
			fmt.Sprintf("mac=%s", generateMACForKVM(n.Name)),
		}
		if n.UEFI {
			// disable iPXE boot
			devParams = append(devParams, "romfile=")
		}
		params = append(params, "-device", strings.Join(devParams, ","))
	}

	if n.UEFI {
		p := r.nvramPath(n.Name)
		err := createNVRAM(ctx, p)
		if err != nil {
			log.Error("Failed to create nvram", map[string]interface{}{
				"error": err,
			})
			return nil, err
		}
	}

	if r.enableVirtFS {
		p := path.Join(r.sharedDir, n.Name)
		err := os.MkdirAll(p, 0755)
		if err != nil {
			return nil, err
		}
		params = append(params, "-virtfs", fmt.Sprintf("local,path=%s,mount_tag=placemat,security_model=none", p))
	}

	params = append(params, "-boot", fmt.Sprintf("reboot-timeout=%d", int64(defaultRebootTimeout/time.Millisecond)))

	guest := r.guestSocketPath(n.Name)
	params = append(params, "-chardev", "socket,id=char0,path="+guest+",server,nowait")
	params = append(params, "-device", "virtio-serial")
	params = append(params, "-device", "virtserialport,chardev=char0,name=placemat")

	monitor := r.monitorSocketPath(n.Name)
	params = append(params, "-monitor", "unix:"+monitor+",server,nowait")

	log.Info("Starting VM", map[string]interface{}{"name": n.Name})
	qemuCommand := well.CommandContext(ctx, "qemu-system-x86_64", params...)
	qemuCommand.Stdout = newColoredLogWriter("qemu", n.Name, os.Stdout)
	qemuCommand.Stderr = newColoredLogWriter("qemu", n.Name, os.Stderr)

	err := qemuCommand.Start()
	if err != nil {
		return nil, err
	}

	for {
		_, err := os.Stat(monitor)
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}

		_, err2 := os.Stat(guest)
		if err2 != nil && !os.IsNotExist(err2) {
			return nil, err2
		}

		if err == nil && err2 == nil {
			break
		}

		select {
		case <-time.After(100 * time.Millisecond):
		case <-ctx.Done():
			return nil, nil
		}
	}

	connGuest, err := net.Dial("unix", guest)
	if err != nil {
		return nil, err
	}
	gc := &guestConnection{
		serial: n.SMBIOS.Serial,
		guest:  connGuest,
		ch:     nodeCh,
	}
	go gc.Handle()

	cleanup := func() {
		connGuest.Close()
		os.Remove(guest)
		os.Remove(monitor)
		os.Remove(r.socketPath(n.Name))
	}

	vm := &NodeVM{
		cmd:     qemuCommand,
		monitor: monitor,
		running: true,
		cleanup: cleanup,
	}

	return vm, err
}

func generateMACForKVM(name string) string {
	vendorPrefix := "52:54:00" // QEMU's vendor prefix
	bytes := sha1.Sum([]byte(name))
	return fmt.Sprintf("%s:%02x:%02x:%02x", vendorPrefix, bytes[0], bytes[1], bytes[2])
}

func createNVRAM(ctx context.Context, p string) error {
	_, err := os.Stat(p)
	if !os.IsNotExist(err) {
		return nil
	}
	return well.CommandContext(ctx, "cp", defaultOVMFVarsPath, p).Run()
}
