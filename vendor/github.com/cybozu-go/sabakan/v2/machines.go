package sabakan

import (
	"errors"
	"regexp"
	"time"

	version "github.com/hashicorp/go-version"
)

// MachineState represents a machine's state.
type MachineState string

// String implements fmt.Stringer interface.
func (ms MachineState) String() string {
	return string(ms)
}

// IsValid returns true only if the MachineState is pre-defined.
func (ms MachineState) IsValid() bool {
	switch ms {
	case StateUninitialized:
		return true
	case StateHealthy:
		return true
	case StateUnhealthy:
		return true
	case StateUnreachable:
		return true
	case StateUpdating:
		return true
	case StateRetiring:
		return true
	case StateRetired:
		return true
	}

	return false
}

// Machine state definitions.
const (
	StateUninitialized = MachineState("uninitialized")
	StateHealthy       = MachineState("healthy")
	StateUnhealthy     = MachineState("unhealthy")
	StateUnreachable   = MachineState("unreachable")
	StateUpdating      = MachineState("updating")
	StateRetiring      = MachineState("retiring")
	StateRetired       = MachineState("retired")
)

var (
	reValidBmcType       = regexp.MustCompile(`^[a-z0-9A-Z-_/.]+$`)
	reValidLabelName     = regexp.MustCompile(`^[a-z0-9A-Z]([a-z0-9A-Z_.-]{0,61}[a-z0-9A-Z])?$`)
	reValidLabelVal      = regexp.MustCompile(`^[a-z0-9A-Z]([a-z0-9A-Z_.-]{0,61}[a-z0-9A-Z])?$`)
	permittedTransitions = map[MachineState][]MachineState{
		StateUninitialized: {StateHealthy, StateRetiring},
		StateHealthy:       {StateUnhealthy, StateUnreachable, StateUpdating, StateRetiring},
		StateUnhealthy:     {StateHealthy, StateUnreachable, StateRetiring},
		StateUnreachable:   {StateHealthy, StateRetiring},
		StateUpdating:      {StateUninitialized},
		StateRetiring:      {StateRetired},
		StateRetired:       {StateUninitialized},
	}
)

// IsValidRole returns true if role is valid as machine role
func IsValidRole(role string) bool {
	return reValidLabelVal.MatchString(role)
}

// IsValidIgnitionID returns true if id is valid as ignition ID
func IsValidIgnitionID(id string) bool {
	_, err := version.NewVersion(id)
	return err == nil
}

// IsValidBmcType returns true if role is valid as BMC type
func IsValidBmcType(bmcType string) bool {
	return reValidBmcType.MatchString(bmcType)
}

// IsValidLabelName returns true if label name is valid
// This is the same as the validation for Kubernetes label names.
func IsValidLabelName(name string) bool {
	return reValidLabelName.MatchString(name)
}

// IsValidLabelValue returns true if label value is valid
// This is the same as the validation for Kubernetes label values.
func IsValidLabelValue(value string) bool {
	if value == "" {
		return true
	}
	return reValidLabelVal.MatchString(value)
}

// MachineBMC is a bmc interface struct for Machine
type MachineBMC struct {
	IPv4 string `json:"ipv4"`
	IPv6 string `json:"ipv6"`
	Type string `json:"type"`
}

// MachineSpec is a set of attributes to define a machine.
type MachineSpec struct {
	Serial       string            `json:"serial"`
	Labels       map[string]string `json:"labels"`
	Rack         uint              `json:"rack"`
	IndexInRack  uint              `json:"index-in-rack"`
	Role         string            `json:"role"`
	IPv4         []string          `json:"ipv4"`
	IPv6         []string          `json:"ipv6"`
	RegisterDate time.Time         `json:"register-date"`
	RetireDate   time.Time         `json:"retire-date"`
	BMC          MachineBMC        `json:"bmc"`
}

// MachineStatus represents the status of a machine.
type MachineStatus struct {
	Timestamp time.Time    `json:"timestamp"`
	Duration  float64      `json:"duration"`
	State     MachineState `json:"state"`
}

// NetworkInfo represents NIC configurations.
type NetworkInfo struct {
	IPv4 []NICConfig `json:"ipv4"`
}

// BMCInfo represents BMC NIC configuration information.
type BMCInfo struct {
	IPv4 NICConfig `json:"ipv4"`
}

// NICConfig represents NIC configuration information.
type NICConfig struct {
	Address  string `json:"address"`
	Netmask  string `json:"netmask"`
	MaskBits int    `json:"maskbits"`
	Gateway  string `json:"gateway"`
}

// MachineInfo is a set of associated information of a Machine.
type MachineInfo struct {
	Network NetworkInfo `json:"network"`
	BMC     BMCInfo     `json:"bmc"`
}

// Machine represents a server hardware.
type Machine struct {
	Spec   MachineSpec   `json:"spec"`
	Status MachineStatus `json:"status"`
	Info   MachineInfo   `json:"info"`
}

// NewMachine creates a new machine instance.
func NewMachine(spec MachineSpec) *Machine {
	return &Machine{
		Spec: spec,
		Status: MachineStatus{
			Timestamp: time.Now().UTC(),
			State:     StateUninitialized,
		},
	}
}

func (m *Machine) isPermittedTransition(to MachineState) bool {
	for _, v := range permittedTransitions[m.Status.State] {
		if v == to {
			return true
		}
	}
	return false
}

// SetState sets the state of the machine.
func (m *Machine) SetState(ms MachineState) error {
	if m.Status.State == ms {
		return nil
	}

	_, ok := permittedTransitions[m.Status.State]
	if !ok {
		return errors.New(m.Status.State.String() + " has no permitted states")
	}

	if !m.isPermittedTransition(ms) {
		return errors.New("transition from " + m.Status.State.String() + " to state " + ms.String() + " is forbidden")
	}

	m.Status.State = ms
	m.Status.Timestamp = time.Now().UTC()
	return nil
}

// AddLabels adds labels to Machine by merging maps.
func (m *Machine) AddLabels(labels map[string]string) {
	if m.Spec.Labels == nil {
		m.Spec.Labels = make(map[string]string)
	}

	for k, v := range labels {
		m.Spec.Labels[k] = v
	}
}

// DeleteLabel deletes label from Machine.
func (m *Machine) DeleteLabel(label string) error {
	_, ok := m.Spec.Labels[label]
	if !ok {
		return ErrNotFound
	}

	delete(m.Spec.Labels, label)
	return nil
}
