package placemat

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cybozu-go/well"
)

// NodeVolumeSpec represents a Node's Volume specification in YAML
type NodeVolumeSpec struct {
	Kind          string `json:"kind"`
	Name          string `json:"name"`
	Image         string `json:"image,omitempty"`
	UserData      string `json:"user-data,omitempty"`
	NetworkConfig string `json:"network-config,omitempty"`
	Size          string `json:"size,omitempty"`
	Folder        string `json:"folder,omitempty"`
	CopyOnWrite   bool   `json:"copy-on-write,omitempty"`
	Cache         string `json:"cache,omitempty"`
	Format        string `json:"format,omitempty"`
	VG            string `json:"vg,omitempty"`
}

// NodeVolume defines the interface for Node volumes.
type NodeVolume interface {
	Kind() string
	Name() string
	Resolve(*Cluster) error
	Create(context.Context, string) ([]string, error)
}

const (
	nodeVolumeCacheWriteback    = "writeback"
	nodeVolumeCacheNone         = "none"
	nodeVolumeCacheWritethrough = "writethrough"
	nodeVolumeCacheDirectSync   = "directsync"
	nodeVolumeCacheUnsafe       = "unsafe"

	nodeVolumeFormatQcow2 = "qcow2"
	nodeVolumeFormatRaw   = "raw"
)

func selectAIOforCache(cache string) string {
	if cache == nodeVolumeCacheNone {
		return "native"
	}
	return "threads"
}

type baseVolume struct {
	name  string
	cache string
}

func (v baseVolume) Name() string {
	return v.name
}

func volumePath(dataDir, name string) string {
	return filepath.Join(dataDir, name+".img")
}

func (v baseVolume) qemuArgs(p string) []string {
	return []string{
		"-drive",
		fmt.Sprintf("if=virtio,cache=%s,aio=%s,file=%s", v.cache, selectAIOforCache(v.cache), p),
	}
}

type imageVolume struct {
	baseVolume
	imageName   string
	image       *Image
	copyOnWrite bool
}

// NewImageVolume creates a volume for type "image".
func NewImageVolume(name string, cache string, imageName string, cow bool) NodeVolume {
	return &imageVolume{
		baseVolume:  baseVolume{name: name, cache: cache},
		imageName:   imageName,
		copyOnWrite: cow,
	}
}

func (v imageVolume) Kind() string {
	return "image"
}

func (v *imageVolume) Resolve(c *Cluster) error {
	img, err := c.GetImage(v.imageName)
	if err != nil {
		return err
	}
	v.image = img
	return nil
}

func (v *imageVolume) Create(ctx context.Context, dataDir string) ([]string, error) {
	p := volumePath(dataDir, v.name)
	args := v.qemuArgs(p)

	_, err := os.Stat(p)
	if err == nil {
		return args, nil
	}

	if !os.IsNotExist(err) {
		return nil, err
	}

	if v.image.File != "" {
		fp, err := filepath.Abs(v.image.File)
		if err != nil {
			return nil, err
		}
		if v.copyOnWrite {
			err = createCoWImageFromBase(ctx, fp, p)
			if err != nil {
				return nil, err
			}
		} else {
			err = writeToFile(fp, p, v.image.decomp)
			if err != nil {
				return nil, err
			}
		}
		return args, nil
	}

	baseImage := v.image.Path()
	if v.copyOnWrite {
		err = createCoWImageFromBase(ctx, baseImage, p)
		if err != nil {
			return nil, err
		}
		return args, nil
	}

	f, err := os.Open(baseImage)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	g, err := os.Create(p)
	if err != nil {
		return nil, err
	}
	defer g.Close()

	_, err = io.Copy(g, f)
	if err != nil {
		return nil, err
	}
	return args, nil
}

func createCoWImageFromBase(ctx context.Context, base, dest string) error {
	c := well.CommandContext(ctx, "qemu-img", "create", "-f", "qcow2", "-o", "backing_file="+base, dest)
	return c.Run()
}

type localDSVolume struct {
	baseVolume
	userData      string
	networkConfig string
}

// NewLocalDSVolume creates a volume for type "localds".
func NewLocalDSVolume(name, cache string, u, n string) NodeVolume {
	return &localDSVolume{
		baseVolume:    baseVolume{name: name, cache: cache},
		userData:      u,
		networkConfig: n,
	}
}

func (v localDSVolume) Kind() string {
	return "localds"
}

func (v *localDSVolume) Resolve(c *Cluster) error {
	return nil
}

func (v *localDSVolume) Create(ctx context.Context, dataDir string) ([]string, error) {
	p := volumePath(dataDir, v.name)

	_, err := os.Stat(p)
	switch {
	case os.IsNotExist(err):
		if v.networkConfig == "" {
			err := well.CommandContext(ctx, "cloud-localds", p, v.userData).Run()
			if err != nil {
				return nil, err
			}
		} else {
			err := well.CommandContext(ctx, "cloud-localds", p, v.userData, "--network-config", v.networkConfig).Run()
			if err != nil {
				return nil, err
			}
		}
	case err == nil:
	default:
		return nil, err
	}

	return v.qemuArgs(p), nil
}

type rawVolume struct {
	baseVolume
	size   string
	format string
}

// NewRawVolume creates a volume for type "raw".
func NewRawVolume(name string, cache string, size, format string) NodeVolume {
	return &rawVolume{
		baseVolume: baseVolume{name: name, cache: cache},
		size:       size,
		format:     format,
	}
}

func (v rawVolume) Kind() string {
	return "raw"
}

func (v *rawVolume) Resolve(c *Cluster) error {
	return nil
}

func (v *rawVolume) Create(ctx context.Context, dataDir string) ([]string, error) {
	p := volumePath(dataDir, v.name)
	_, err := os.Stat(p)
	switch {
	case os.IsNotExist(err):
		err = well.CommandContext(ctx, "qemu-img", "create", "-f", v.format, p, v.size).Run()
		if err != nil {
			return nil, err
		}
	case err == nil:
	default:
		return nil, err
	}
	return v.qemuArgs(p), nil
}

func (v *rawVolume) qemuArgs(p string) []string {
	return []string{
		"-drive",
		fmt.Sprintf("if=virtio,cache=%s,aio=%s,format=%s,file=%s", v.cache, selectAIOforCache(v.cache), v.format, p),
	}
}

type lvVolume struct {
	baseVolume
	size string
	vg   string
}

// NewLVVolume creates a volume for type "lv".
func NewLVVolume(name string, cache string, size, vg string) NodeVolume {
	return &lvVolume{
		baseVolume: baseVolume{name: name, cache: cache},
		size:       size,
		vg:         vg,
	}
}

func (v lvVolume) Kind() string {
	return "lv"
}

func (v *lvVolume) Resolve(c *Cluster) error {
	return nil
}

func (v *lvVolume) Create(ctx context.Context, dataDir string) ([]string, error) {
	nodeName := filepath.Base(dataDir)
	lvName := nodeName + "." + v.name

	output, err := exec.Command("lvs", "--noheadings", "--unbuffered", "-o", "lv_name", v.vg).Output()
	if err != nil {
		return nil, err
	}

	found := false
	for _, line := range strings.Split(string(output), "\n") {
		if strings.TrimSpace(line) == lvName {
			found = true
		}
	}
	if !found {
		err := exec.Command("lvcreate", "-n", lvName, "-L", v.size, v.vg).Run()
		if err != nil {
			return nil, err
		}
	}

	output, err = exec.Command("lvs", "--noheadings", "--unbuffered", "-o", "lv_path", v.vg+"/"+lvName).Output()
	if err != nil {
		return nil, err
	}
	p := strings.TrimSpace(string(output))

	return v.qemuArgs(p), nil
}

func (v *lvVolume) qemuArgs(p string) []string {
	return []string{
		"-drive",
		fmt.Sprintf("if=virtio,cache=%s,aio=%s,format=raw,file=%s", v.cache, selectAIOforCache(v.cache), p),
	}
}

type vvfatVolume struct {
	baseVolume
	folderName string
	folder     *DataFolder
}

// NewVVFATVolume creates a volume for type "vvfat".
func NewVVFATVolume(name string, folderName string) NodeVolume {
	return &vvfatVolume{
		baseVolume: baseVolume{name: name},
		folderName: folderName,
	}
}

func (v vvfatVolume) Kind() string {
	return "vvfat"
}

func (v *vvfatVolume) Resolve(c *Cluster) error {
	df, err := c.GetDataFolder(v.folderName)
	if err != nil {
		return err
	}
	v.folder = df
	return nil
}

func (v *vvfatVolume) Create(ctx context.Context, _ string) ([]string, error) {
	return v.qemuArgs(v.folder.Path()), nil
}

func (v vvfatVolume) qemuArgs(p string) []string {
	return []string{
		"-drive",
		"file=fat:16:" + p + ",format=raw,if=virtio",
	}
}
