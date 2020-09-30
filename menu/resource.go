package menu

import (
	"net"
)

// NodeType represent node type(i.g. boot, CP, CS, SS)
type NodeType int

const (
	// BootNode represent node type of boot server
	BootNode NodeType = iota
	// CPNode represent node type of control plane
	CPNode
	// CSNode represent node type of compute server
	CSNode
	// SSNode represent node type of storage server
	SSNode
)

// NetworkMenu represents network settings to be written to the configuration file
type NetworkMenu struct {
	IPAMConfigFile string
	NodeBase       net.IP
	NodeRangeSize  int
	NodeRangeMask  int
	BMC            *net.IPNet
	ASNBase        int
	Internet       *net.IPNet
	CoreSpine      *net.IPNet
	CoreExternal   *net.IPNet
	CoreOperation  *net.IPNet
	SpineTor       net.IP
	Proxy          net.IP
	NTP            []net.IP
	Pod            *net.IPNet
	Bastion        *net.IPNet
	LoadBalancer   *net.IPNet
	Ingress        *net.IPNet
	Global         *net.IPNet
}

// InventoryMenu represents inventory settings to be written to the configuration file
type InventoryMenu struct {
	ClusterID string
	Spine     int
	Rack      []RackMenu
}

// RackMenu represents how many nodes each rack contains
type RackMenu struct {
	CP int
	CS int
	SS int
}

// NodeMenu represents computing resources used by each type nodes
type NodeMenu struct {
	Type              NodeType
	CPU               int
	Memory            string
	Image             string
	Data              []string
	UEFI              bool
	CloudInitTemplate string
	TPM               bool
}

// Menu is a top-level structure that summarizes the settings of each menus
type Menu struct {
	Network   *NetworkMenu
	Inventory *InventoryMenu
	Images    []*imageSpec
	Nodes     []*NodeMenu
}
