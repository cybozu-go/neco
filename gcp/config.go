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
	defaultHomeDiskSizeGB = 50
	// DefaultPreemptible is default value for enabling preemptible
	// https://cloud.google.com/compute/docs/instances/preemptible
	defaultPreemptible = false
	// defaultAppShutdownAt is default time for test instance auto-shutdown
	defaultAppShutdownAt = "20:00"
	// DefaultShutdownAt is default time for instance auto-shutdown
	defaultShutdownAt = "21:00"
	// DefaultTimeZone is default timezone for instance auto-shutdown
	defaultTimeZone = "Asia/Tokyo"
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
	Stop            []string      `yaml:"stop"`
	Exclude         []string      `yaml:"exclude"`
	Expiration      time.Duration `yaml:"expiration"`
	Timezone        string        `yaml:"timezone"`
	ShutdownAt      string        `yaml:"shutdown-at"`
	AdditionalZones []string      `yaml:"additional-zones"`
}

// ComputeConfig is configuration for GCE
type ComputeConfig struct {
	MachineType      string             `yaml:"machine-type"`
	BootDiskSizeGB   int                `yaml:"boot-disk-sizeGB"`
	OptionalPackages []string           `yaml:"optional-packages"`
	HostVM           HostVMConfig       `yaml:"host-vm"`
	AutoShutdown     AutoShutdownConfig `yaml:"auto-shutdown"`

	// backward compatibility
	VMXEnabled struct {
		OptionalPackages []string `yaml:"optional-packages"`
	} `yaml:"vmx-enabled"`
}

// HostVMConfig is configuration for host-vm instance
type HostVMConfig struct {
	HomeDisk       bool `yaml:"home-disk"`
	HomeDiskSizeGB int  `yaml:"home-disk-sizeGB"`
	Preemptible    bool `yaml:"preemptible"`
}

// AutoShutdownConfig is configuration for automatically shutting down host-vm instance
type AutoShutdownConfig struct {
	Timezone   string `yaml:"timezone"`
	ShutdownAt string `yaml:"shutdown-at"`
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
				Timezone:   defaultTimeZone,
				ShutdownAt: defaultAppShutdownAt,
			},
		},
		Compute: ComputeConfig{
			BootDiskSizeGB: defaultBootDiskSizeGB,
			AutoShutdown: AutoShutdownConfig{
				Timezone:   defaultTimeZone,
				ShutdownAt: defaultShutdownAt,
			},
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
			Zone:           "asia-northeast2-c",
		},
		App: AppConfig{
			Shutdown: ShutdownConfig{
				Exclude: []string{
					"neco-ops",
					"neco-apps-release",
					"neco-apps-master",
				},
				Expiration: 2 * time.Hour,
				Timezone:   defaultTimeZone,
				ShutdownAt: defaultAppShutdownAt,
				AdditionalZones: []string{
					"asia-northeast1-c",
				},
			},
		},
		Compute: ComputeConfig{
			MachineType:    "n1-standard-32",
			BootDiskSizeGB: defaultBootDiskSizeGB,
			HostVM: HostVMConfig{
				HomeDisk:       defaultHomeDisk,
				HomeDiskSizeGB: defaultHomeDiskSizeGB,
				Preemptible:    defaultPreemptible,
			},
		},
	}
}
