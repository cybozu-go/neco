package sabakan

import (
	"encoding/json"
)

// IgnitionVersion represents the specification version of Ignition.
type IgnitionVersion string

// Supported ignition versions
const (
	Ignition2_2 = IgnitionVersion("2.2")
	Ignition2_3 = IgnitionVersion("2.3")
)

// IgnitionTemplate represents an ignition template.
// The exact type of Template is determined by Version.
type IgnitionTemplate struct {
	Version  IgnitionVersion        `json:"version"`
	Template json.RawMessage        `json:"template"`
	Metadata map[string]interface{} `json:"meta"`
}
