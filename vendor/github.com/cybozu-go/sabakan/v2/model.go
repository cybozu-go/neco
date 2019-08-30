package sabakan

import (
	"context"
	"errors"
	"io"
	"net"
	"time"
)

// ErrConflicted is a special error for models.
// A model should return this when it fails to update a resource due to conflicts.
var ErrConflicted = errors.New("key conflicted")

// ErrNotFound is a special err for models.
// A model should return this when it cannot find a resource by a specified key.
var ErrNotFound = errors.New("not found")

// ErrBadRequest is a special err for models.
// A model should return this when the request is bad
var ErrBadRequest = errors.New("bad request")

// ErrEncryptionKeyExists is a special err for models.
// A model should return this when encryption key exists.
var ErrEncryptionKeyExists = errors.New("encryption key exists")

// StorageModel is an interface for disk encryption keys.
type StorageModel interface {
	GetEncryptionKey(ctx context.Context, serial string, diskByPath string) ([]byte, error)
	PutEncryptionKey(ctx context.Context, serial string, diskByPath string, key []byte) error
	DeleteEncryptionKeys(ctx context.Context, serial string) ([]string, error)
}

// MachineModel is an interface for machine database.
type MachineModel interface {
	Register(ctx context.Context, machines []*Machine) error
	Get(ctx context.Context, serial string) (*Machine, error)
	SetState(ctx context.Context, serial string, state MachineState) error
	PutLabel(ctx context.Context, serial string, label, value string) error
	DeleteLabel(ctx context.Context, serial string, label string) error
	SetRetireDate(ctx context.Context, serial string, date time.Time) error
	Query(ctx context.Context, query Query) ([]*Machine, error)
	Delete(ctx context.Context, serial string) error
}

// IPAMModel is an interface for IPAMConfig.
type IPAMModel interface {
	PutConfig(ctx context.Context, config *IPAMConfig) error
	GetConfig() (*IPAMConfig, error)
}

// DHCPModel is an interface for DHCPConfig.
type DHCPModel interface {
	PutConfig(ctx context.Context, config *DHCPConfig) error
	GetConfig() (*DHCPConfig, error)
	Lease(ctx context.Context, ifaddr net.IP, mac net.HardwareAddr) (net.IP, error)
	Renew(ctx context.Context, ciaddr net.IP, mac net.HardwareAddr) error
	Release(ctx context.Context, ciaddr net.IP, mac net.HardwareAddr) error
	Decline(ctx context.Context, ciaddr net.IP, mac net.HardwareAddr) error
}

// ImageModel is an interface to manage boot images.
type ImageModel interface {
	// These are for /api/v1/images
	GetIndex(ctx context.Context, os string) (ImageIndex, error)
	Upload(ctx context.Context, os, id string, r io.Reader) error
	Download(ctx context.Context, os, id string, out io.Writer) error
	Delete(ctx context.Context, os, id string) error

	// This is for /api/v1/boot/OS/{kernel,initrd.gz}
	// Calling f will serve the content to the HTTP client.
	ServeFile(ctx context.Context, os, filename string,
		f func(modtime time.Time, content io.ReadSeeker)) error
}

// AssetHandler is an interface for AssetModel.Get
type AssetHandler interface {
	ServeContent(asset *Asset, content io.ReadSeeker)
	Redirect(url string)
}

// AssetModel is an interface to manage assets.
type AssetModel interface {
	GetIndex(ctx context.Context) ([]string, error)
	GetInfo(ctx context.Context, name string) (*Asset, error)
	Put(ctx context.Context, name, contentType string, csum []byte, options map[string]string, r io.Reader) (*AssetStatus, error)
	Get(ctx context.Context, name string, h AssetHandler) error
	Delete(ctx context.Context, name string) error
}

// IgnitionModel is an interface for ignition template.
type IgnitionModel interface {
	PutTemplate(ctx context.Context, role, id string, tmpl *IgnitionTemplate) error
	GetTemplateIDs(ctx context.Context, role string) ([]string, error)
	GetTemplate(ctx context.Context, role string, id string) (*IgnitionTemplate, error)
	DeleteTemplate(ctx context.Context, role string, id string) error
}

// LogModel is an interface for audit logs.
type LogModel interface {
	Dump(ctx context.Context, since, until time.Time, w io.Writer) error
}

// KernelParamsModel is an interface for kernel parameters.
type KernelParamsModel interface {
	PutParams(ctx context.Context, os string, params string) error
	GetParams(ctx context.Context, os string) (string, error)
}

// HealthModel is an interface for etcd health status
type HealthModel interface {
	GetHealth(ctx context.Context) error
}

// SchemaModel is an interface for schema versioning.
type SchemaModel interface {
	Version(ctx context.Context) (string, error)
	Upgrade(ctx context.Context) error
}

// Runner is an interface to run the underlying goroutines.
//
// The caller must pass a channel as follows.
// Receiving a value from the channel effectively guarantees that
// the driver gets ready.
//
//    ch := make(chan struct{})
//    well.Go(func(ctx context.Context) error {
//        driver.Run(ctx, ch)
//    })
//    <-ch
type Runner interface {
	Run(ctx context.Context, ch chan<- struct{}) error
}

// Model is a struct that consists of sub-models.
type Model struct {
	Runner
	Storage      StorageModel
	Machine      MachineModel
	IPAM         IPAMModel
	DHCP         DHCPModel
	Image        ImageModel
	Asset        AssetModel
	Ignition     IgnitionModel
	Log          LogModel
	KernelParams KernelParamsModel
	Health       HealthModel
	Schema       SchemaModel
}
