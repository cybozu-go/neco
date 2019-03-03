package web

// SMBIOSStatus represents SMBIOS of a Node
type SMBIOSStatus struct {
	Manufacturer string `json:"manufacturer"`
	Product      string `json:"product"`
	Serial       string `json:"serial"`
}

// NodeStatus represents status of a Node
type NodeStatus struct {
	Name       string            `json:"name"`
	Taps       map[string]string `json:"taps"`
	Volumes    []string          `json:"volumes"`
	CPU        int               `json:"cpu"`
	Memory     string            `json:"memory"`
	UEFI       bool              `json:"uefi"`
	SMBIOS     SMBIOSStatus      `json:"smbios"`
	IsRunning  bool              `json:"is_running"`
	SocketPath string            `json:"socket_path"`
}

// PodStatus represents status of a Pod
type PodStatus struct {
	Name    string            `json:"name"`
	PID     int               `json:"pid"`
	UUID    string            `json:"uuid"`
	Veths   map[string]string `json:"veths"`
	Volumes []string          `json:"volumes"`
	Apps    []string          `json:"apps"`
}
