package sabakan

import (
	"errors"
	"regexp"
	"time"
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
	reValidRole      = regexp.MustCompile(`^[a-zA-Z][0-9a-zA-Z._-]*$`)
	reValidBmcType   = regexp.MustCompile(`^[a-z0-9A-Z-_/.]+$`)
	reValidLabelVal  = regexp.MustCompile(`^[[:print:]]+$`)
	reValidLabelName = regexp.MustCompile(`^[a-z0-9A-Z-_/.]+$`)
)

// IsValidRole returns true if role is valid as machine role
func IsValidRole(role string) bool {
	return reValidRole.MatchString(role)
}

// IsValidBmcType returns true if role is valid as BMC type
func IsValidBmcType(bmcType string) bool {
	return reValidBmcType.MatchString(bmcType)
}

// IsValidLabelName returns true if label name is valid
func IsValidLabelName(name string) bool {
	return reValidLabelName.MatchString(name)
}

// IsValidLabelValue returns true if label value is valid
func IsValidLabelValue(value string) bool {
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

// Machine represents a server hardware.
type Machine struct {
	Spec   MachineSpec   `json:"spec"`
	Status MachineStatus `json:"status"`
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

// SetState sets the state of the machine.
func (m *Machine) SetState(ms MachineState) error {
	if m.Status.State == ms {
		return nil
	}

	switch m.Status.State {
	case StateUninitialized:
		if ms != StateHealthy && ms != StateRetiring {
			return errors.New("transition to state other than healthy or retiring is forbidden")
		}
	case StateHealthy:
		if ms == StateUninitialized || ms == StateRetired {
			return errors.New("transition to " + ms.String() + " is forbidden")
		}
	case StateUnhealthy:
		if ms != StateUninitialized && ms != StateRetiring {
			return errors.New("transition to " + ms.String() + " is forbidden")
		}
	case StateUnreachable:
		if ms == StateUpdating || ms == StateRetired || ms == StateUnhealthy {
			return errors.New("transition to " + ms.String() + " is forbidden")
		}
	case StateUpdating:
		if ms != StateUninitialized {
			return errors.New("transition to state other than uninitialized is forbidden")
		}
	case StateRetiring:
		if ms != StateRetired {
			return errors.New("transition to state other than retired is forbidden")
		}
	case StateRetired:
		if ms != StateUninitialized {
			return errors.New("transition to state other than uninitialized is forbidden")
		}
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
