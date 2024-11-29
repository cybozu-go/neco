package menu

import (
	"crypto/sha1"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/cybozu-go/netutil"
	"github.com/cybozu-go/placemat/v2/pkg/types"
)

const (
	offsetInternetHost = 1
	offsetInternetCore = 2

	offsetExternalCore = 1
	offsetExternal     = 2

	offsetOperationCore = 1
	offsetOperation     = 2

	offsetNodenetToR     = 1
	offsetNodenetBoot    = 3
	offsetNodenetServers = 4

	offsetASNCore     = -3
	offsetASNExternal = -2
	offsetASNSpine    = -1

	offsetBMCHost = 1
	offsetBMCCore = 2

	offsetNTP1    = 1
	offsetNTP2    = 2
	offsetNTPCore = 3
)

// Cluster is the config file generator for Placemat
type Cluster struct {
	network       *network
	operation     *operation
	external      *external
	internet      *internet
	bmc           *bmc
	core          *core
	spines        []*spine
	racks         []*rack
	image         []*types.ImageSpec
	deviceclasses []*types.DeviceClassSpec
}

// GenerateOption is a option for Cluster.Generate().
type GenerateOption struct {
	ChronyTag string
}

type network struct {
	ipamConfigFile string
	nodeBase       net.IP
	nodeRangeSize  int
	nodeRangeMask  int
	bmc            *net.IPNet
	asnBase        int
	asnCore        int
	asnExternal    int
	asnSpine       int
	internet       *net.IPNet
	coreSpine      *net.IPNet
	coreExternal   *net.IPNet
	coreOperation  *net.IPNet
	spineTor       net.IP
	proxy          net.IP
	ntp            *net.IPNet
	pod            *net.IPNet
	bastion        *net.IPNet
	loadBalancer   *net.IPNet
	egress         *net.IPNet
	global         *net.IPNet
}

type rack struct {
	name                  string
	shortName             string
	index                 int
	asn                   int
	nodeNetworkPrefixSize int
	tor1                  *tor
	tor2                  *tor
	bootNode              *bootNode
	csList                []*node
	ssList                []*node
	ss2List               []*node
	node0Network          *net.IPNet
	node1Network          *net.IPNet
	node2Network          *net.IPNet
}

type node struct {
	name         string
	fullName     string
	spec         *nodeSpec
	serial       string
	node0Address *net.IPNet
	node1Address *net.IPNet
	node2Address *net.IPNet
	tor1Address  *net.IPNet
	tor2Address  *net.IPNet
}

type bootNode struct {
	node
	bastionAddress *net.IPNet
}

type core struct {
	internetAddress  *net.IPNet
	spineAddresses   []*net.IPNet
	operationAddress *net.IPNet
	externalAddress  *net.IPNet
	proxyAddress     net.IP
	coreNTPAddress   *net.IPNet
	ntpServers       []*net.IPNet
}

type spine struct {
	name         string
	shortName    string
	coreAddress  *net.IPNet
	torAddresses []*net.IPNet
	bmcAddress   *net.IPNet
}

func (s spine) tor1Address(rackIdx int) *net.IPNet {
	return s.torAddresses[rackIdx*2]
}

func (s spine) tor2Address(rackIdx int) *net.IPNet {
	return s.torAddresses[rackIdx*2+1]
}

type tor struct {
	name           string
	spineAddresses []*net.IPNet
	nodeAddress    *net.IPNet
	nodeInterface  string
}

type operation struct {
	coreAddress *net.IPNet
}

func (o *operation) netns() *types.NetNSSpec {
	return &types.NetNSSpec{
		Kind: "NetworkNamespace",
		Name: "operation",
		Interfaces: []*types.NetNSInterfaceSpec{
			{
				Network:   "core-to-op",
				Addresses: []string{o.coreAddress.String()},
			},
		},
		InitScripts: []string{"setup-default-gateway-operation"},
	}
}

type external struct {
	coreAddress *net.IPNet
}

func (o *external) netns() *types.NetNSSpec {
	return &types.NetNSSpec{
		Kind: "NetworkNamespace",
		Name: "external",
		Interfaces: []*types.NetNSInterfaceSpec{
			{
				Network:   "core-to-ext",
				Addresses: []string{o.coreAddress.String()},
			},
		},
		InitScripts: []string{"setup-default-gateway-external"},
	}
}

type internet struct {
	address *net.IPNet
}

func (i *internet) network() *types.NetworkSpec {
	return &types.NetworkSpec{
		Kind:    "Network",
		Name:    "internet",
		Type:    "external",
		UseNAT:  true,
		Address: i.address.String(),
	}
}

type bmc struct {
	address *net.IPNet
}

func (b *bmc) network() *types.NetworkSpec {
	return &types.NetworkSpec{
		Kind:    "Network",
		Name:    "bmc",
		Type:    "bmc",
		UseNAT:  false,
		Address: b.address.String(),
	}
}

// NewCluster creates a Cluster instance
func NewCluster(menu *menu) (*Cluster, error) {
	network := menu.network
	inventory := menu.inventory.Spec

	cluster := &Cluster{
		network:  network,
		internet: &internet{address: addToIPNet(network.internet, offsetInternetHost)},
		bmc:      &bmc{address: addToIPNet(network.bmc, offsetBMCHost)},
		core: &core{
			internetAddress:  addToIPNet(network.internet, offsetInternetCore),
			operationAddress: addToIPNet(network.coreOperation, offsetOperationCore),
			externalAddress:  addToIPNet(network.coreExternal, offsetExternalCore),
			proxyAddress:     network.proxy,
			coreNTPAddress:   addToIPNet(network.ntp, offsetNTPCore),
			ntpServers: []*net.IPNet{
				addToIPNet(network.ntp, offsetNTP1),
				addToIPNet(network.ntp, offsetNTP2),
			},
		},
		operation:     &operation{coreAddress: addToIPNet(network.coreOperation, offsetOperation)},
		external:      &external{coreAddress: addToIPNet(network.coreExternal, offsetExternal)},
		image:         menu.images,
		deviceclasses: menu.deviceclasses,
	}

	// Set racks
	nodeSpecs := make(map[nodeType]*nodeSpec)
	for _, node := range menu.nodes {
		nodeSpecs[node.Type] = &node.Spec
	}

	var tors []*tor
	for rackIdx, rackMenu := range inventory.Rack {
		node0Network := makeNodeNetwork(network.nodeBase, network.nodeRangeSize, network.nodeRangeMask, rackIdx*3+0)
		node1Network := makeNodeNetwork(network.nodeBase, network.nodeRangeSize, network.nodeRangeMask, rackIdx*3+1)
		node2Network := makeNodeNetwork(network.nodeBase, network.nodeRangeSize, network.nodeRangeMask, rackIdx*3+2)

		rack := &rack{
			name:                  fmt.Sprintf("rack%d", rackIdx),
			shortName:             fmt.Sprintf("r%d", rackIdx),
			index:                 rackIdx,
			asn:                   network.asnBase + rackIdx,
			nodeNetworkPrefixSize: network.nodeRangeMask,
			bootNode: &bootNode{
				node: node{
					name:         "boot",
					fullName:     fmt.Sprintf("boot-%d", rackIdx),
					spec:         nodeSpecs[nodeTypeBoot],
					serial:       fmt.Sprintf("%x", sha1.Sum([]byte(fmt.Sprintf("boot-%d", rackIdx)))),
					node0Address: addToIP(node0Network.IP, offsetNodenetBoot, 32),
					node1Address: addToIPNet(node1Network, offsetNodenetBoot),
					node2Address: addToIPNet(node2Network, offsetNodenetBoot),
					tor1Address:  addToIPNet(node1Network, offsetNodenetToR),
					tor2Address:  addToIPNet(node2Network, offsetNodenetToR),
				},
				bastionAddress: addToIP(network.bastion.IP, rackIdx, 32),
			},
			node0Network: node0Network,
			node1Network: node1Network,
			node2Network: node2Network,
		}

		for csIdx := 0; csIdx < rackMenu.Cs; csIdx++ {
			name := fmt.Sprintf("cs-%d", csIdx+1)
			fullName := fmt.Sprintf("%s-%s", rack.name, name)
			node := &node{
				name:         name,
				fullName:     fullName,
				spec:         nodeSpecs[nodeTypeCS],
				serial:       fmt.Sprintf("%x", sha1.Sum([]byte(fullName))),
				node0Address: addToIP(node0Network.IP, offsetNodenetServers+csIdx, 32),
				node1Address: addToIPNet(node1Network, offsetNodenetServers+csIdx),
				node2Address: addToIPNet(node2Network, offsetNodenetServers+csIdx),
				tor1Address:  rack.bootNode.tor1Address,
				tor2Address:  rack.bootNode.tor2Address,
			}
			rack.csList = append(rack.csList, node)
		}

		for ssIdx := 0; ssIdx < rackMenu.Ss; ssIdx++ {
			name := fmt.Sprintf("ss-%d", ssIdx+1)
			fullName := fmt.Sprintf("%s-%s", rack.name, name)
			node := &node{
				name:         name,
				fullName:     fullName,
				spec:         nodeSpecs[nodeTypeSS],
				serial:       fmt.Sprintf("%x", sha1.Sum([]byte(fullName))),
				node0Address: addToIP(node0Network.IP, offsetNodenetServers+ssIdx+rackMenu.Cs, 32),
				node1Address: addToIPNet(node1Network, offsetNodenetServers+ssIdx+rackMenu.Cs),
				node2Address: addToIPNet(node2Network, offsetNodenetServers+ssIdx+rackMenu.Cs),
				tor1Address:  rack.bootNode.tor1Address,
				tor2Address:  rack.bootNode.tor2Address,
			}
			rack.ssList = append(rack.ssList, node)
		}

		for ss2Idx := 0; ss2Idx < rackMenu.Ss2; ss2Idx++ {
			name := fmt.Sprintf("ss2-%d", ss2Idx+1)
			fullName := fmt.Sprintf("%s-%s", rack.name, name)
			node := &node{
				name:         name,
				fullName:     fullName,
				spec:         nodeSpecs[nodeTypeSS2],
				serial:       fmt.Sprintf("%x", sha1.Sum([]byte(fullName))),
				node0Address: addToIP(node0Network.IP, offsetNodenetServers+ss2Idx+rackMenu.Cs+rackMenu.Ss, 32),
				node1Address: addToIPNet(node1Network, offsetNodenetServers+ss2Idx+rackMenu.Cs+rackMenu.Ss),
				node2Address: addToIPNet(node2Network, offsetNodenetServers+ss2Idx+rackMenu.Cs+rackMenu.Ss),
				tor1Address:  rack.bootNode.tor1Address,
				tor2Address:  rack.bootNode.tor2Address,
			}
			rack.ss2List = append(rack.ss2List, node)
		}

		rack.tor1 = &tor{
			name:          fmt.Sprintf("rack%d-tor%d", rackIdx, 1),
			nodeAddress:   addToIPNet(rack.node1Network, offsetNodenetToR),
			nodeInterface: fmt.Sprintf("eth%d", inventory.Spine),
		}
		tors = append(tors, rack.tor1)

		rack.tor2 = &tor{
			name:          fmt.Sprintf("rack%d-tor%d", rackIdx, 2),
			nodeAddress:   addToIPNet(rack.node2Network, offsetNodenetToR),
			nodeInterface: fmt.Sprintf("eth%d", inventory.Spine),
		}
		tors = append(tors, rack.tor2)

		cluster.racks = append(cluster.racks, rack)
	}

	// Set spines and ip addresses for core, spine and ToR
	coreSpineBaseAddr := network.coreSpine
	spineTorBaseAddr := &net.IPNet{IP: network.spineTor, Mask: net.CIDRMask(31, 32)}
	for spineIdx := 0; spineIdx < inventory.Spine; spineIdx++ {
		cluster.core.spineAddresses = append(cluster.core.spineAddresses, coreSpineBaseAddr)
		coreSpineBaseAddr = addToIPNet(coreSpineBaseAddr, 1)
		spine := &spine{
			name:         fmt.Sprintf("spine%d", spineIdx+1),
			shortName:    fmt.Sprintf("s%d", spineIdx+1),
			coreAddress:  coreSpineBaseAddr,
			torAddresses: nil,
			bmcAddress:   addToIPNet(network.bmc, spineIdx+2),
		}
		coreSpineBaseAddr = addToIPNet(coreSpineBaseAddr, 1)

		for _, tor := range tors {
			spine.torAddresses = append(spine.torAddresses, spineTorBaseAddr)
			spineTorBaseAddr = addToIPNet(spineTorBaseAddr, 1)

			tor.spineAddresses = append(tor.spineAddresses, spineTorBaseAddr)
			spineTorBaseAddr = addToIPNet(spineTorBaseAddr, 1)
		}

		cluster.spines = append(cluster.spines, spine)
	}

	return cluster, nil
}

// Generate generates config files for Placemat
func (c *Cluster) Generate(inputDir, outputDir string, opt *GenerateOption) error {
	f, err := os.Create(filepath.Join(outputDir, "cluster.yml"))
	if err != nil {
		return err
	}

	sabakanDir, err := filepath.Abs(filepath.Join(outputDir, sabakanDir))
	if err != nil {
		return err
	}
	if err := c.generateClusterYaml(f, sabakanDir); err != nil {
		return err
	}

	return c.generateConfFiles(inputDir, outputDir, opt)
}

func parseNetworkCIDR(s string) (net.IP, *net.IPNet, error) {
	ip, network, err := net.ParseCIDR(s)
	if err != nil {
		return nil, nil, err
	}
	if !ip.Equal(network.IP) {
		return nil, nil, fmt.Errorf("host part of network address must be 0: %s", s)
	}
	return ip, network, nil
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
