package gcp

import "time"

const (
	// DefaultExpiration is default expiration time
	defaultExpiration = "0s"
	// DefaultBootDiskSizeGB is default instance boot disk size
	defaultBootDiskSizeGB = 20
	// DefaultHomeDisk is default value for attaching home disk image in host-vm
	defaultHomeDisk = false
	// DefaultHomeDiskSizeGB is default home disk size
	defaultHomeDiskSizeGB = 20
	// DefaultPreemptible is default value for enabling preemptible
	// https://cloud.google.com/compute/docs/instances/preemptible
	defaultPreemptible            = false
	defaultVmxEnabledImage        = "debian-9-stretch-v20180911"
	defaultVmxEnabledImageProject = "debian-cloud"
)

// Config is configuration for necogcp command and GAE app
type Config struct {
	Common  CommonConfig  `yaml:"common"`
	App     AppConfig     `yaml:"app"`
	Compute ComputeConfig `yaml:"compute"`
}

// CommonConfig is common configuration for GCP
type CommonConfig struct {
	Project        string `yaml:"project"`
	ServiceAccount string `yaml:"serviceaccount"`
	Zone           string `yaml:"zone"`
}

// AppConfig is configuration for GAE app
type AppConfig struct {
	Shutdown ShutdownConfig `yaml:"shutdown"`
}

// ShutdownConfig is automatic shutdown configuration
type ShutdownConfig struct {
	Stop       []string      `yaml:"stop"`
	Exclude    []string      `yaml:"exclude"`
	Expiration time.Duration `yaml:"expiration"`
}

// ComputeConfig is configuration for GCE
type ComputeConfig struct {
	MachineType    string           `yaml:"machine-type"`
	BootDiskSizeGB int              `yaml:"boot-disk-sizeGB"`
	VMXEnabled     VMXEnabledConfig `yaml:"vmx-enabled"`
	HostVM         HostVMConfig     `yaml:"host-vm"`
}

// VMXEnabledConfig is configuration for vmx-enabled image
type VMXEnabledConfig struct {
	Image            string   `yaml:"image"`
	ImageProject     string   `yaml:"image-project"`
	OptionalPackages []string `yaml:"optional-packages"`
}

// HostVMConfig is configuration for host-vm instance
type HostVMConfig struct {
	HomeDisk       bool `yaml:"home-disk"`
	HomeDiskSizeGB int  `yaml:"home-disk-sizeGB"`
	Preemptible    bool `yaml:"preemptible"`
}

// NewConfig returns Config
func NewConfig() (*Config, error) {
	expiration, err := time.ParseDuration(defaultExpiration)
	if err != nil {
		return nil, err
	}

	return &Config{
		App: AppConfig{
			Shutdown: ShutdownConfig{
				Expiration: expiration,
			},
		},
		Compute: ComputeConfig{
			BootDiskSizeGB: defaultBootDiskSizeGB,
			HostVM: HostVMConfig{
				HomeDisk:       defaultHomeDisk,
				HomeDiskSizeGB: defaultHomeDiskSizeGB,
				Preemptible:    defaultPreemptible,
			},
		},
	}, nil
}

// NecoTestConfig returns configuration for neco-test
func NecoTestConfig() *Config {
	return &Config{
		Common: CommonConfig{
			Project:        "neco-test",
			ServiceAccount: "neco-test@neco-test.iam.gserviceaccount.com",
			Zone:           "asia-northeast1-c",
		},
		App: AppConfig{
			Shutdown: ShutdownConfig{
				Exclude: []string{
					"neco-ops",
				},
				Expiration: 3600 * time.Second,
			},
		},
		Compute: ComputeConfig{
			MachineType:    "n1-highcpu-64",
			BootDiskSizeGB: defaultBootDiskSizeGB,
			HostVM: HostVMConfig{
				HomeDisk:       defaultHomeDisk,
				HomeDiskSizeGB: defaultHomeDiskSizeGB,
				Preemptible:    defaultPreemptible,
			},
			VMXEnabled: VMXEnabledConfig{
				Image:        defaultVmxEnabledImage,
				ImageProject: defaultVmxEnabledImageProject,
			},
		},
	}
}
