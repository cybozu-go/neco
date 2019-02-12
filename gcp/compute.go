package gcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
)

const (
	retryCount   = 300
	imageLicense = "https://www.googleapis.com/compute/v1/projects/vm-options/global/licenses/enable-vmx"
)

// ComputeClient is GCP compute client using "gcloud compute"
type ComputeClient struct {
	cfg      *Config
	instance string
	user     string
	image    string
}

// NewComputeClient returns ComputeClient
func NewComputeClient(cfg *Config, instance string) *ComputeClient {
	user := os.Getenv("USER")
	if cfg.Common.Project == "neco-test" {
		user = "cybozu"
	}

	image := fmt.Sprintf("%s-%s", cfg.Compute.VMXEnabled.Image, instance)

	return &ComputeClient{
		cfg:      cfg,
		instance: instance,
		user:     user,
		image:    image,
	}
}

func (cc *ComputeClient) gCloudCompute() []string {
	return []string{"echo", "gcloud", "--account", cc.cfg.Common.ServiceAccount, "--project", cc.cfg.Common.Project, "compute"}
}

func (cc *ComputeClient) gCloudComputeInstances() []string {
	return []string{"echo", "gcloud", "--account", cc.cfg.Common.ServiceAccount, "--project", cc.cfg.Common.Project, "compute", "instances"}
}

func (cc *ComputeClient) gCloudComputeImages() []string {
	return []string{"echo", "gcloud", "--account", cc.cfg.Common.ServiceAccount, "--project", cc.cfg.Common.Project, "compute", "images"}
}

func (cc *ComputeClient) gCloudComputeDisks() []string {
	return []string{"echo", "gcloud", "--account", cc.cfg.Common.ServiceAccount, "--project", cc.cfg.Common.Project, "compute", "disks"}
}

func (cc *ComputeClient) gCloudComputeSSH(command []string) []string {
	return []string{"echo", "gcloud", "--account", cc.cfg.Common.ServiceAccount, "--project", cc.cfg.Common.Project, "compute", "ssh",
		fmt.Sprintf("%s@%s", cc.user, cc.instance),
		fmt.Sprintf("--command=\"%s\"", strings.Join(command, " "))}
}

// CreateVMXEnabledInstance creates vmx-enabled instance
func (cc *ComputeClient) CreateVMXEnabledInstance(ctx context.Context) error {
	gcmd := cc.gCloudComputeInstances()
	gcmd = append(gcmd, "create", cc.instance,
		"--zone", cc.cfg.Common.Zone,
		"--image", cc.cfg.Compute.VMXEnabled.Image,
		"--image-project", cc.cfg.Compute.VMXEnabled.ImageProject,
		"--boot-disk-type", "pd-ssd",
		"--boot-disk-size", cc.cfg.Compute.BootDiskSize,
		"--machine-type", cc.cfg.Compute.MachineType)
	c := exec.CommandContext(ctx, gcmd[0], gcmd[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// CreateHostVMInstance creates host-vm instance
func (cc *ComputeClient) CreateHostVMInstance(ctx context.Context) error {
	gcmd := cc.gCloudComputeInstances()
	gcmd = append(gcmd, "create", cc.instance,
		"--zone", cc.cfg.Common.Zone,
		"--image", cc.image,
		"--boot-disk-type", "pd-ssd",
		"--boot-disk-size", cc.cfg.Compute.BootDiskSize,
		"--local-ssd", "interface=scsi",
		"--machine-type", cc.cfg.Compute.MachineType,
		"--scopes", "https://www.googleapis.com/auth/devstorage.read_write")
	if cc.cfg.Compute.HostVM.Preemptible {
		gcmd = append(gcmd, "--preemptible")
	}
	c := exec.CommandContext(ctx, gcmd[0], gcmd[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// CreateHomeDisk creates home disk image
func (cc *ComputeClient) CreateHomeDisk(ctx context.Context) error {
	if !cc.cfg.Compute.HostVM.HomeDisk {
		return nil
	}
	gcmdInfo := cc.gCloudComputeDisks()
	gcmdInfo = append(gcmdInfo, "describe", "home", "--format", "json")
	outBuf := new(bytes.Buffer)
	c := exec.CommandContext(ctx, gcmdInfo[0], gcmdInfo[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = outBuf
	c.Stderr = os.Stderr
	err := c.Run()
	if err == nil {
		log.Info("gcp: home disk already exists", nil)
		return nil
	}

	configSize := cc.cfg.Compute.HostVM.HomeDiskSize
	gcmdCreate := cc.gCloudComputeDisks()
	gcmdCreate = append(gcmdCreate, "create", "home", "--size", configSize)
	c = exec.CommandContext(ctx, gcmdCreate[0], gcmdCreate[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// AttachHomeDisk attaches home disk image to host-vm instance
func (cc *ComputeClient) AttachHomeDisk(ctx context.Context) error {
	if !cc.cfg.Compute.HostVM.HomeDisk {
		return nil
	}
	gcmd := cc.gCloudComputeInstances()
	gcmd = append(gcmd, "attach-disk", cc.instance,
		"--zone", cc.cfg.Common.Zone,
		"--disk", "home",
		"--device-name", "home")
	if cc.cfg.Compute.HostVM.Preemptible {
		gcmd = append(gcmd, "--preemptible")
	}
	c := exec.CommandContext(ctx, gcmd[0], gcmd[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// ResizeHomeDisk resizes home disk image
func (cc *ComputeClient) ResizeHomeDisk(ctx context.Context) error {
	if !cc.cfg.Compute.HostVM.HomeDisk {
		return nil
	}
	gcmdInfo := cc.gCloudComputeDisks()
	gcmdInfo = append(gcmdInfo, "describe", "home", "--format", "json")
	outBuf := new(bytes.Buffer)
	c := exec.CommandContext(ctx, gcmdInfo[0], gcmdInfo[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = outBuf
	c.Stderr = os.Stderr
	err := c.Run()
	if err != nil {
		return err
	}

	var info map[string]interface{}
	err = json.Unmarshal(outBuf.Bytes(), info)
	if err != nil {
		return err
	}

	currentSize := info["sizeGB"].(string)
	configSize := cc.cfg.Compute.HostVM.HomeDiskSize
	if currentSize != cc.cfg.Compute.HostVM.HomeDiskSize {
		log.Error("gcp: current home disk size is smaller or same as size in configuration file", map[string]interface{}{
			"currentSize": currentSize,
			"configSize":  configSize,
		})
		return nil
	}

	gcmdResize := cc.gCloudComputeDisks()
	gcmdResize = append(gcmdResize, "resize", "home", "--size", configSize)
	c = exec.CommandContext(ctx, gcmdResize[0], gcmdResize[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// DeleteInstance deletes given instance
func (cc *ComputeClient) DeleteInstance(ctx context.Context) error {
	gcmd := cc.gCloudComputeInstances()
	gcmd = append(gcmd, "delete", cc.instance, "--zone", cc.cfg.Common.Zone, "--quiet")
	c := exec.CommandContext(ctx, gcmd[0], gcmd[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// WaitInstance waits given instance until online
func (cc *ComputeClient) WaitInstance(ctx context.Context) error {
	gcmd := cc.gCloudComputeSSH([]string{"date"})
	c := exec.CommandContext(ctx, gcmd[0], gcmd[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	return neco.RetryWithSleep(ctx, retryCount, time.Second,
		func(ctx context.Context) error {
			return c.Run()
		},
		func(err error) {
			log.Error("gcp: failed to check online of the instance", map[string]interface{}{
				log.FnError: err,
				"instance":  cc.instance,
			})
		},
	)
}

// StopInstance stops given instance
func (cc *ComputeClient) StopInstance(ctx context.Context) error {
	gcmd := cc.gCloudComputeInstances()
	gcmd = append(gcmd, "stop", cc.instance,
		"--zone", cc.cfg.Common.Zone,
		"--quiet")
	c := exec.CommandContext(ctx, gcmd[0], gcmd[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// CreateVMXEnabledImage create GCE vmx-enabled image
func (cc *ComputeClient) CreateVMXEnabledImage(ctx context.Context) error {
	gcmd := cc.gCloudComputeImages()
	gcmd = append(gcmd, "create", cc.image,
		"--source-disk", cc.instance,
		"--source-disk-zone", cc.cfg.Common.Zone,
		"--licenses", imageLicense,
		"--quiet")
	c := exec.CommandContext(ctx, gcmd[0], gcmd[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// DeleteVMXEnabledImage create GCE vmx-enabled image
func (cc *ComputeClient) DeleteVMXEnabledImage(ctx context.Context) error {
	gcmd := cc.gCloudComputeImages()
	gcmd = append(gcmd, "delete", cc.image, "--quiet")
	c := exec.CommandContext(ctx, gcmd[0], gcmd[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// Upload uploads a file to the instance through ssh
func (cc *ComputeClient) Upload(ctx context.Context, file string) error {
	gcmd := cc.gCloudComputeInstances()
	gcmd = append(gcmd, "scp", file, fmt.Sprintf("%s@%s:/tmp", cc.user, cc.instance))
	c := exec.CommandContext(ctx, gcmd[0], gcmd[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// RunSetup executes "necogcp setup" on the instance through ssh
func (cc *ComputeClient) RunSetup(ctx context.Context, progFile, cfgFile string) error {
	err := cc.Upload(ctx, progFile)
	if err != nil {
		return err
	}

	err = cc.Upload(ctx, cfgFile)
	if err != nil {
		return err
	}

	gcmd := cc.gCloudComputeSSH([]string{"sudo", "/tmp/" + filepath.Base(progFile), "--config", "/tmp/" + filepath.Base(cfgFile), "setup"})
	c := exec.CommandContext(ctx, gcmd[0], gcmd[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
