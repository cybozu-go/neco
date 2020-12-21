package menu

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"net"

	"github.com/cybozu-go/netutil"
)

const (
	torPerRack = 2

	offsetInternetHost = 1
	offsetInternetCore = 2

	offsetExternalCore     = 1
	offsetExternalExternal = 2

	offsetOperationCore      = 1
	offsetOperationOperation = 2

	offsetNodenetToR     = 1
	offsetNodenetBoot    = 3
	offsetNodenetServers = 4

	offsetASNCore     = -3
	offsetASNExternal = -2
	offsetASNSpine    = -1

	offsetBMCHost = 1
	offsetBMCCore = 2
)

// Rack is template args for rack
type Rack struct {
	Name                  string
	ShortName             string
	Index                 int
	ASN                   int
	NodeNetworkPrefixSize int
	ToR1                  ToR
	ToR2                  ToR
	BootNode              BootNodeEntity
	CSList                []Node
	SSList                []Node
	node0Network          *net.IPNet
	node1Network          *net.IPNet
	node2Network          *net.IPNet
}

// Node is a template args for a node
type Node struct {
	Name         string
	Fullname     string // some func compose full name by itself...
	Serial       string
	Node0Address *net.IPNet
	Node1Address *net.IPNet
	Node2Address *net.IPNet
	ToR1Address  *net.IPNet
	ToR2Address  *net.IPNet
}

// ToR is a template args for a ToR switch
type ToR struct {
	Name           string
	SpineAddresses []*net.IPNet
	NodeAddress    *net.IPNet
	NodeInterface  string
}

// BootNodeEntity is a template args for a boot node
type BootNodeEntity struct {
	Node

	BastionAddress *net.IPNet
}

// Spine is a template args for Spine
type Spine struct {
	Name         string
	ShortName    string
	CoreAddress  *net.IPNet
	ToRAddresses []*net.IPNet
}

// ToR1Address returns spine's IP address connected from ToR-1 in the specified rack
func (s Spine) ToR1Address(rackIdx int) *net.IPNet {
	return s.ToRAddresses[rackIdx*2]
}

// ToR2Address returns spine's IP address connected from ToR-2 in the specified rack
func (s Spine) ToR2Address(rackIdx int) *net.IPNet {
	return s.ToRAddresses[rackIdx*2+1]
}

// Endpoints contains endpoints for external hosts
type Endpoints struct {
	Host      *net.IPNet
	External  *net.IPNet
	Operation *net.IPNet
}

// Core contains parameters to construct core router
type Core struct {
	InternetAddress  *net.IPNet
	BMCAddress       *net.IPNet
	SpineAddresses   []*net.IPNet
	OperationAddress *net.IPNet
	ExternalAddress  *net.IPNet
	ProxyAddress     net.IP
	NTPAddresses     []net.IP
}

// TemplateArgs is args for cluster.yml
type TemplateArgs struct {
	Network struct {
		Exposed struct {
			Bastion      *net.IPNet
			LoadBalancer *net.IPNet
			Ingress      *net.IPNet
			Global       *net.IPNet
		}
		BMC         *net.IPNet
		Endpoints   Endpoints
		ASNExternal int
		ASNSpine    int
		ASNCore     int
		Pod         *net.IPNet
	}
	ClusterID string
	Racks     []Rack
	Spines    []Spine
	Core      Core
	CS        VMResource
	SS        VMResource
	Boot      VMResource
	Images    []*imageSpec
}

// BIRDRackTemplateArgs is args to generate bird config for each rack
type BIRDRackTemplateArgs struct {
	Args    TemplateArgs
	RackIdx int
}

// BIRDSpineTemplateArgs is args to generate bird config for each spine
type BIRDSpineTemplateArgs struct {
	Args     TemplateArgs
	SpineIdx int
}

// VMResource is args to specify vm resource
type VMResource struct {
	CPU               int
	Memory            string
	Image             string
	Data              []string
	UEFI              bool
	CloudInitTemplate string
	TPM               bool
}

// ToTemplateArgs is converter Menu to TemplateArgs
func ToTemplateArgs(menu *Menu) (*TemplateArgs, error) {
	var templateArgs TemplateArgs

	setNetworkArgs(&templateArgs, menu)

	templateArgs.Images = menu.Images

	definedImages := map[string]bool{}
	for _, image := range menu.Images {
		definedImages[image.Name] = true
	}

	for _, node := range menu.Nodes {
		switch node.Type {
		case CSNode:
			templateArgs.CS.Memory = node.Memory
			templateArgs.CS.CPU = node.CPU
			templateArgs.CS.Image = node.Image
			templateArgs.CS.Data = node.Data
			templateArgs.CS.UEFI = node.UEFI
			templateArgs.CS.CloudInitTemplate = node.CloudInitTemplate
			templateArgs.CS.TPM = node.TPM
		case SSNode:
			templateArgs.SS.Memory = node.Memory
			templateArgs.SS.CPU = node.CPU
			templateArgs.SS.Image = node.Image
			templateArgs.SS.Data = node.Data
			templateArgs.SS.UEFI = node.UEFI
			templateArgs.SS.CloudInitTemplate = node.CloudInitTemplate
			templateArgs.SS.TPM = node.TPM
		case BootNode:
			templateArgs.Boot.Memory = node.Memory
			templateArgs.Boot.CPU = node.CPU
			templateArgs.Boot.Image = node.Image
			templateArgs.Boot.Data = node.Data
			templateArgs.Boot.UEFI = node.UEFI
			templateArgs.Boot.CloudInitTemplate = node.CloudInitTemplate
			templateArgs.Boot.TPM = node.TPM
		default:
			return nil, errors.New("invalid node type")
		}

		requiredImages := node.Data
		if len(node.Image) > 0 {
			requiredImages = append(requiredImages, node.Image)
		}
		for _, img := range requiredImages {
			if !definedImages[img] {
				return nil, errors.New("no such Image resource: " + node.Image)
			}
		}
	}

	templateArgs.ClusterID = menu.Inventory.ClusterID

	numRack := len(menu.Inventory.Rack)

	spineToRackBases := make([][]net.IP, menu.Inventory.Spine)
	for spineIdx := 0; spineIdx < menu.Inventory.Spine; spineIdx++ {
		spineToRackBases[spineIdx] = make([]net.IP, numRack)
		for rackIdx := range menu.Inventory.Rack {
			offset := int64((spineIdx*numRack + rackIdx) * torPerRack * 2)
			spineToRackBases[spineIdx][rackIdx] = netutil.IPAdd(menu.Network.SpineTor, offset)
		}
	}

	templateArgs.Racks = make([]Rack, numRack)
	for rackIdx, rackMenu := range menu.Inventory.Rack {
		rack := &templateArgs.Racks[rackIdx]
		rack.Name = fmt.Sprintf("rack%d", rackIdx)
		rack.Index = rackIdx
		rack.ShortName = fmt.Sprintf("r%d", rackIdx)
		rack.ASN = menu.Network.ASNBase + rackIdx
		rack.node0Network = makeNodeNetwork(menu.Network.NodeBase, menu.Network.NodeRangeSize, menu.Network.NodeRangeMask, rackIdx*3+0)
		rack.node1Network = makeNodeNetwork(menu.Network.NodeBase, menu.Network.NodeRangeSize, menu.Network.NodeRangeMask, rackIdx*3+1)
		rack.node2Network = makeNodeNetwork(menu.Network.NodeBase, menu.Network.NodeRangeSize, menu.Network.NodeRangeMask, rackIdx*3+2)

		constructToRAddresses(rack, rackIdx, menu, spineToRackBases)
		buildBootNode(rack, menu)
		rack.NodeNetworkPrefixSize = menu.Network.NodeRangeMask

		for csIdx := 0; csIdx < rackMenu.CS; csIdx++ {
			node := buildNode("cs", csIdx, offsetNodenetServers, rack)
			rack.CSList = append(rack.CSList, node)
		}
		for ssIdx := 0; ssIdx < rackMenu.SS; ssIdx++ {
			node := buildNode("ss", ssIdx, offsetNodenetServers+rackMenu.CS, rack)
			rack.SSList = append(rack.SSList, node)
		}
	}

	templateArgs.Spines = make([]Spine, menu.Inventory.Spine)
	for spineIdx := 0; spineIdx < menu.Inventory.Spine; spineIdx++ {
		spine := &templateArgs.Spines[spineIdx]
		spine.Name = fmt.Sprintf("spine%d", spineIdx+1)
		spine.ShortName = fmt.Sprintf("s%d", spineIdx+1)

		spine.CoreAddress = addToIPNet(menu.Network.CoreSpine, (2*spineIdx)+1)
		// {internet} + {tor per rack} * {rack}
		spine.ToRAddresses = make([]*net.IPNet, torPerRack*numRack)
		for rackIdx := range menu.Inventory.Rack {
			spine.ToRAddresses[rackIdx*torPerRack] = addToIP(spineToRackBases[spineIdx][rackIdx], 0, 31)
			spine.ToRAddresses[rackIdx*torPerRack+1] = addToIP(spineToRackBases[spineIdx][rackIdx], 2, 31)
		}
	}

	setCore(&templateArgs, menu)
	return &templateArgs, nil
}

func setNetworkArgs(templateArgs *TemplateArgs, menu *Menu) {
	templateArgs.Network.BMC = menu.Network.BMC
	templateArgs.Network.ASNCore = menu.Network.ASNBase + offsetASNCore
	templateArgs.Network.ASNExternal = menu.Network.ASNBase + offsetASNExternal
	templateArgs.Network.ASNSpine = menu.Network.ASNBase + offsetASNSpine
	templateArgs.Network.Exposed.Bastion = menu.Network.Bastion
	templateArgs.Network.Exposed.LoadBalancer = menu.Network.LoadBalancer
	templateArgs.Network.Exposed.Ingress = menu.Network.Ingress
	templateArgs.Network.Exposed.Global = menu.Network.Global
	templateArgs.Network.Endpoints.Host = addToIPNet(menu.Network.Internet, offsetInternetHost)
	templateArgs.Network.Endpoints.External = addToIPNet(menu.Network.CoreExternal, offsetExternalExternal)
	templateArgs.Network.Endpoints.Operation = addToIPNet(menu.Network.CoreOperation, offsetOperationOperation)
	templateArgs.Network.Pod = menu.Network.Pod
}

func buildNode(basename string, idx int, offsetStart int, rack *Rack) Node {
	node := Node{}
	node.Name = fmt.Sprintf("%v%d", basename, idx+1)
	node.Fullname = fmt.Sprintf("%s-%s", rack.Name, node.Name)
	node.Serial = fmt.Sprintf("%x", sha1.Sum([]byte(node.Fullname)))
	offset := offsetStart + idx

	node.Node0Address = addToIP(rack.node0Network.IP, offset, 32)
	node.Node1Address = addToIPNet(rack.node1Network, offset)
	node.Node2Address = addToIPNet(rack.node2Network, offset)
	node.ToR1Address = rack.BootNode.ToR1Address
	node.ToR2Address = rack.BootNode.ToR2Address
	return node
}

func buildBootNode(rack *Rack, menu *Menu) {
	rack.BootNode.Name = "boot"
	rack.BootNode.Fullname = fmt.Sprintf("%s-%d", rack.BootNode.Name, rack.Index)
	rack.BootNode.Serial = fmt.Sprintf("%x", sha1.Sum([]byte(rack.BootNode.Fullname)))

	rack.BootNode.Node0Address = addToIP(rack.node0Network.IP, offsetNodenetBoot, 32)
	rack.BootNode.Node1Address = addToIPNet(rack.node1Network, offsetNodenetBoot)
	rack.BootNode.Node2Address = addToIPNet(rack.node2Network, offsetNodenetBoot)
	rack.BootNode.BastionAddress = addToIP(menu.Network.Bastion.IP, rack.Index, 32)

	rack.BootNode.ToR1Address = addToIPNet(rack.node1Network, offsetNodenetToR)
	rack.BootNode.ToR2Address = addToIPNet(rack.node2Network, offsetNodenetToR)
}

func setCore(ta *TemplateArgs, menu *Menu) {
	for i := range ta.Spines {
		ta.Core.SpineAddresses = append(ta.Core.SpineAddresses, addToIPNet(menu.Network.CoreSpine, 2*i))
	}
	ta.Core.BMCAddress = addToIPNet(menu.Network.BMC, offsetBMCCore)
	ta.Core.OperationAddress = addToIPNet(menu.Network.CoreOperation, offsetOperationCore)
	ta.Core.InternetAddress = addToIPNet(menu.Network.Internet, offsetInternetCore)
	ta.Core.ExternalAddress = addToIPNet(menu.Network.CoreExternal, offsetExternalCore)
	ta.Core.ProxyAddress = menu.Network.Proxy
	ta.Core.NTPAddresses = append(ta.Core.NTPAddresses, menu.Network.NTP...)
}

func constructToRAddresses(rack *Rack, rackIdx int, menu *Menu, bases [][]net.IP) {
	rack.ToR1.SpineAddresses = make([]*net.IPNet, menu.Inventory.Spine)
	for spineIdx := 0; spineIdx < menu.Inventory.Spine; spineIdx++ {
		rack.ToR1.SpineAddresses[spineIdx] = addToIP(bases[spineIdx][rackIdx], 1, 31)
	}
	rack.ToR1.NodeAddress = addToIPNet(rack.node1Network, offsetNodenetToR)
	rack.ToR1.NodeInterface = fmt.Sprintf("eth%d", menu.Inventory.Spine)

	rack.ToR2.SpineAddresses = make([]*net.IPNet, menu.Inventory.Spine)
	for spineIdx := 0; spineIdx < menu.Inventory.Spine; spineIdx++ {
		rack.ToR2.SpineAddresses[spineIdx] = addToIP(bases[spineIdx][rackIdx], 3, 31)
	}
	rack.ToR2.NodeAddress = addToIPNet(rack.node2Network, offsetNodenetToR)
	rack.ToR2.NodeInterface = fmt.Sprintf("eth%d", menu.Inventory.Spine)
}

func addToIPNet(netAddr *net.IPNet, offset int) *net.IPNet {
	return &net.IPNet{IP: netutil.IPAdd(netAddr.IP, int64(offset)), Mask: netAddr.Mask}
}

func addToIP(netIP net.IP, offset int, prefixSize int) *net.IPNet {
	mask := net.CIDRMask(prefixSize, 32)
	return &net.IPNet{IP: netutil.IPAdd(netIP, int64(offset)), Mask: mask}
}

func makeNodeNetwork(base net.IP, rangeSize, prefixSize int, nodeIdx int) *net.IPNet {
	offset := 1 << rangeSize * nodeIdx
	return &net.IPNet{IP: netutil.IPAdd(base, int64(offset)), Mask: net.CIDRMask(prefixSize, 32)}
}
