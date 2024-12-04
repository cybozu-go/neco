package menu

import (
	"fmt"
	"io"

	"github.com/cybozu-go/placemat/v2/pkg/types"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"sigs.k8s.io/yaml"
)

func (c *Cluster) generateClusterYaml(w io.Writer, sabakanDir string) error {
	spec := c.generateClusterSpec(sabakanDir)

	f := json.YAMLFramer.NewFrameWriter(w)

	for _, n := range spec.Networks {
		data, err := yaml.Marshal(n)
		if err != nil {
			return err
		}
		_, err = f.Write(data)
		if err != nil {
			return err
		}
	}

	for _, i := range spec.Images {
		data, err := yaml.Marshal(i)
		if err != nil {
			return err
		}
		_, err = f.Write(data)
		if err != nil {
			return err
		}
	}

	for _, d := range spec.DeviceClasses {
		data, err := yaml.Marshal(d)
		if err != nil {
			return err
		}
		_, err = f.Write(data)
		if err != nil {
			return err
		}
	}

	for _, n := range spec.Nodes {
		data, err := yaml.Marshal(n)
		if err != nil {
			return err
		}
		_, err = f.Write(data)
		if err != nil {
			return err
		}
	}

	for _, n := range spec.NetNSs {
		data, err := yaml.Marshal(n)
		if err != nil {
			return err
		}
		_, err = f.Write(data)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Cluster) generateClusterSpec(sabakanDir string) *types.ClusterSpec {
	spec := &types.ClusterSpec{
		Images:        c.image,
		DeviceClasses: c.deviceclasses,
	}
	c.appendNetworks(spec)
	c.appendNodes(spec, sabakanDir)
	c.appendNetworkNamespaces(spec)
	return spec
}

func (c *Cluster) appendNetworks(spec *types.ClusterSpec) {
	// Internet Network
	spec.Networks = append(spec.Networks, c.internet.network())

	// Core Network
	for _, spine := range c.spines {
		spec.Networks = append(spec.Networks, &types.NetworkSpec{
			Kind: "Network",
			Name: fmt.Sprintf("core-to-%s", spine.shortName),
			Type: "internal",
		})
	}
	spec.Networks = append(spec.Networks,
		&types.NetworkSpec{
			Kind: "Network",
			Name: "core-to-ext",
			Type: "internal",
		},
		&types.NetworkSpec{
			Kind: "Network",
			Name: "core-to-op",
			Type: "internal",
		},
	)
	if c.core.proxyAddress != nil {
		spec.Networks = append(spec.Networks,
			&types.NetworkSpec{
				Kind: "Network",
				Name: "core-node0",
				Type: "internal",
			},
		)
	}
	if c.core.coreNTPAddress != nil {
		spec.Networks = append(spec.Networks, &types.NetworkSpec{
			Kind: "Network",
			Name: "core-ntp",
			Type: "internal",
		})
	}

	// BMC Network
	spec.Networks = append(spec.Networks, c.bmc.network())

	// Spine to ToR Network
	for _, spine := range c.spines {
		for _, rack := range c.racks {
			spec.Networks = append(spec.Networks,
				&types.NetworkSpec{
					Kind: "Network",
					Name: fmt.Sprintf("%s-to-%s-1", spine.shortName, rack.shortName),
					Type: "internal",
				},
				&types.NetworkSpec{
					Kind: "Network",
					Name: fmt.Sprintf("%s-to-%s-2", spine.shortName, rack.shortName),
					Type: "internal",
				},
			)
		}
	}

	// Rack Network
	for _, rack := range c.racks {
		spec.Networks = append(spec.Networks,
			&types.NetworkSpec{
				Kind: "Network",
				Name: fmt.Sprintf("%s-node1", rack.shortName),
				Type: "internal",
			},
			&types.NetworkSpec{
				Kind: "Network",
				Name: fmt.Sprintf("%s-node2", rack.shortName),
				Type: "internal",
			},
		)
	}
}

func (c *Cluster) appendNodes(spec *types.ClusterSpec, sabakanDir string) {
	for _, rack := range c.racks {
		bootSpec := rack.bootNode.spec
		bootNode := &types.NodeSpec{
			Kind: "Node",
			Name: rack.bootNode.fullName,
			Interfaces: []string{
				fmt.Sprintf("%s-node1", rack.shortName),
				fmt.Sprintf("%s-node2", rack.shortName),
			},
			CPU:    bootSpec.CPU,
			Memory: bootSpec.Memory,
			NUMA: types.NUMASpec{
				Nodes: bootSpec.NUMA.Nodes,
			},
			UEFI: bootSpec.UEFI,
			TPM:  bootSpec.TPM,
			SMBIOS: types.SMBIOSConfigSpec{
				Serial: rack.bootNode.serial,
			},
		}
		if bootSpec.SMP != nil {
			bootNode.SMP = &types.SMPSpec{
				CPUs:    bootSpec.SMP.CPUs,
				Cores:   bootSpec.SMP.Cores,
				Threads: bootSpec.SMP.Threads,
				Dies:    bootSpec.SMP.Dies,
				Sockets: bootSpec.SMP.Sockets,
			}
		}
		if bootSpec.Image != "" {
			bootNode.Volumes = append(bootNode.Volumes, types.NodeVolumeSpec{
				Kind:        "image",
				Name:        "root",
				Image:       bootSpec.Image,
				CopyOnWrite: true,
				Cache:       "writeback",
			})

			if bootSpec.CloudInitTemplate != "" {
				bootNode.Volumes = append(bootNode.Volumes, types.NodeVolumeSpec{
					Kind:          "localds",
					Name:          "seed",
					UserData:      fmt.Sprintf("seed_%s.yml", bootNode.Name),
					NetworkConfig: "network.yml",
				})
			}
		} else {
			bootNode.Volumes = append(bootNode.Volumes, types.NodeVolumeSpec{
				Kind: "raw",
				Name: "root",
				Size: "30G",
			})
		}
		bootNode.Volumes = append(bootNode.Volumes, types.NodeVolumeSpec{
			Kind: "hostPath",
			Name: "sabakan",
			Path: sabakanDir,
		})
		spec.Nodes = append(spec.Nodes, bootNode)

		for _, cs := range rack.csList {
			spec.Nodes = append(spec.Nodes, createWorkerNode(rack, cs, cs.spec))
		}
		for _, ss := range rack.ssList {
			spec.Nodes = append(spec.Nodes, createWorkerNode(rack, ss, ss.spec))
		}
	}

	spec.Nodes = append(spec.Nodes, createChronyNode())
}

func createWorkerNode(rack *rack, node *node, spec *nodeSpec) *types.NodeSpec {
	nodeSpec := &types.NodeSpec{
		Kind: "Node",
		Name: fmt.Sprintf("%s-%s", rack.name, node.name),
		Interfaces: []string{
			fmt.Sprintf("%s-node1", rack.shortName),
			fmt.Sprintf("%s-node2", rack.shortName),
		},
		CPU:    spec.CPU,
		Memory: spec.Memory,
		NUMA: types.NUMASpec{
			Nodes: spec.NUMA.Nodes,
		},
		UEFI: spec.UEFI,
		TPM:  spec.TPM,
		SMBIOS: types.SMBIOSConfigSpec{
			Serial: node.serial,
		},
	}
	if spec.SMP != nil {
		nodeSpec.SMP = &types.SMPSpec{
			CPUs:    spec.SMP.CPUs,
			Cores:   spec.SMP.Cores,
			Threads: spec.SMP.Threads,
			Dies:    spec.SMP.Dies,
			Sockets: spec.SMP.Sockets,
		}
		nodeSpec.NetworkDeviceQueue = spec.SMP.CPUs * 2
	} else {
		nodeSpec.NetworkDeviceQueue = spec.CPU * 2
	}

	diskSize := "50G"
	if spec.DiskSize != "" {
		diskSize = spec.DiskSize
	}
	for i := 0; i < spec.DiskCount; i++ {
		nodeSpec.Volumes = append(nodeSpec.Volumes, types.NodeVolumeSpec{
			Kind:   "raw",
			Name:   fmt.Sprintf("data%d", i+1),
			Size:   diskSize,
			Cache:  "none",
			Format: "raw",
		})
	}

	for i, v := range spec.Disks {
		diskSizeDC := diskSize
		if v.Size != "" {
			diskSizeDC = v.Size
		}
		for j := 0; j < v.Count; j++ {
			nodeSpec.Volumes = append(nodeSpec.Volumes, types.NodeVolumeSpec{
				Kind:        "raw",
				Name:        fmt.Sprintf("data%d-%d", i+1, j+1),
				Size:        diskSizeDC,
				Cache:       "none",
				Format:      "raw",
				DeviceClass: v.DeviceClass,
			})
		}
	}

	for i, dataImg := range spec.Data {
		nodeSpec.Volumes = append(nodeSpec.Volumes, types.NodeVolumeSpec{
			Kind:        "image",
			Name:        fmt.Sprintf("extra%d", i),
			Image:       dataImg,
			CopyOnWrite: true,
		})
	}

	return nodeSpec
}

func createChronyNode() *types.NodeSpec {
	return &types.NodeSpec{
		Kind:         "Node",
		Name:         "chrony",
		Interfaces:   []string{"core-ntp"},
		CPU:          2,
		Memory:       "4G",
		TPM:          false,
		IgnitionFile: "chrony.ign",
		Volumes: []types.NodeVolumeSpec{
			{
				Kind:        "image",
				Name:        "root",
				Image:       "flatcar",
				CopyOnWrite: true,
				Cache:       "writeback",
			},
		},
	}
}

func (c *Cluster) appendNetworkNamespaces(spec *types.ClusterSpec) {
	// Core NetworkNamespace
	core := &types.NetNSSpec{
		Kind: "NetworkNamespace",
		Name: "core",
		Interfaces: []*types.NetNSInterfaceSpec{
			{
				Network:   "internet",
				Addresses: []string{c.core.internetAddress.String()},
			},
			{
				Network:   "core-to-ext",
				Addresses: []string{c.core.externalAddress.String()},
			},
			{
				Network:   "core-to-op",
				Addresses: []string{c.core.operationAddress.String()},
			},
		},
		Apps: []*types.NetNSAppSpec{
			{
				Name: "bird",
				Command: []string{
					"/usr/sbin/bird",
					"-f",
					"-c",
					"/etc/bird/bird_core.conf",
					"-s",
					"/var/run/bird/bird_core.ctl",
				},
			},
			{
				Name: "squid",
				Command: []string{
					"/usr/sbin/squid",
					"-N",
				},
			},
		},
		InitScripts: []string{"setup-iptables"},
	}
	if c.core.proxyAddress != nil {
		core.Interfaces = append(core.Interfaces, &types.NetNSInterfaceSpec{
			Network:   "core-node0",
			Addresses: []string{fmt.Sprintf("%s/32", c.core.proxyAddress.String())},
		})
	}
	if c.core.coreNTPAddress != nil {
		core.Interfaces = append(core.Interfaces, &types.NetNSInterfaceSpec{
			Network:   "core-ntp",
			Addresses: []string{c.core.coreNTPAddress.String()},
		})
	}
	for i, spine := range c.core.spineAddresses {
		core.Interfaces = append(core.Interfaces, &types.NetNSInterfaceSpec{
			Network:   fmt.Sprintf("core-to-s%d", i+1),
			Addresses: []string{spine.String()},
		})
	}
	spec.NetNSs = append(spec.NetNSs, core)

	// Spine NetworkNamespace
	for i, spine := range c.spines {
		spineNetNs := &types.NetNSSpec{
			Kind: "NetworkNamespace",
			Name: fmt.Sprintf("spine%d", i+1),
			Interfaces: []*types.NetNSInterfaceSpec{
				{
					Network:   fmt.Sprintf("core-to-%s", spine.shortName),
					Addresses: []string{spine.coreAddress.String()},
				},
			},
			Apps: []*types.NetNSAppSpec{
				{
					Name: "bird",
					Command: []string{
						"/usr/sbin/bird",
						"-f",
						"-c",
						fmt.Sprintf("/etc/bird/bird_%s.conf", spine.name),
						"-s",
						fmt.Sprintf("/var/run/bird/bird_%s.ctl", spine.name),
					},
				},
			},
			InitScripts: []string{"setup-iptables-spine"},
		}

		spineNetNs.Interfaces = append(spineNetNs.Interfaces, &types.NetNSInterfaceSpec{
			Network:   "bmc",
			Addresses: []string{spine.bmcAddress.String()},
		})

		for i, rack := range c.racks {
			spineNetNs.Interfaces = append(spineNetNs.Interfaces, &types.NetNSInterfaceSpec{
				Network:   fmt.Sprintf("%s-to-%s-1", spine.shortName, rack.shortName),
				Addresses: []string{spine.tor1Address(i).String()},
			})

			spineNetNs.Interfaces = append(spineNetNs.Interfaces, &types.NetNSInterfaceSpec{
				Network:   fmt.Sprintf("%s-to-%s-2", spine.shortName, rack.shortName),
				Addresses: []string{spine.tor2Address(i).String()},
			})
		}

		spec.NetNSs = append(spec.NetNSs, spineNetNs)
	}

	// ToR NetworkNamespace
	for _, rack := range c.racks {
		spec.NetNSs = append(spec.NetNSs, c.createToRNetNs(rack, rack.tor1, 1))
		spec.NetNSs = append(spec.NetNSs, c.createToRNetNs(rack, rack.tor2, 2))
	}

	// External NetworkNamespace
	spec.NetNSs = append(spec.NetNSs, c.external.netns())

	// Operation NetworkNamespace
	spec.NetNSs = append(spec.NetNSs, c.operation.netns())
}

func (c *Cluster) createToRNetNs(rack *rack, tor *tor, torIdx int) *types.NetNSSpec {
	name := fmt.Sprintf("%s-tor%d", rack.name, torIdx)
	torNs := &types.NetNSSpec{
		Kind: "NetworkNamespace",
		Name: name,
		Apps: []*types.NetNSAppSpec{
			{
				Name: "bird",
				Command: []string{
					"/usr/sbin/bird",
					"-f",
					"-c",
					fmt.Sprintf("/etc/bird/bird_%s.conf", name),
					"-s",
					fmt.Sprintf("/var/run/bird/bird_%s.ctl", name),
				},
			},
		},
	}
	for i, spine := range c.spines {
		torNs.Interfaces = append(torNs.Interfaces, &types.NetNSInterfaceSpec{
			Network:   fmt.Sprintf("%s-to-%s-%d", spine.shortName, rack.shortName, torIdx),
			Addresses: []string{tor.spineAddresses[i].String()},
		})
	}
	torNs.Interfaces = append(torNs.Interfaces, &types.NetNSInterfaceSpec{
		Network:   fmt.Sprintf("%s-node%d", rack.shortName, torIdx),
		Addresses: []string{tor.nodeAddress.String()},
	})

	dnsmasqCommand := []string{
		"/usr/sbin/dnsmasq",
		"--keep-in-foreground",
		fmt.Sprintf("--pid-file=/var/run/dnsmasq_%s.pid", name),
		"--log-facility=-",
	}

	dnsmasqCommand = append(dnsmasqCommand, "--dhcp-relay", fmt.Sprintf("%s,%s", tor.nodeAddress.IP.String(), "10.71.0.1"))
	dnsmasqCommand = append(dnsmasqCommand, "--dhcp-relay", fmt.Sprintf("%s,%s", tor.nodeAddress.IP.String(), "10.71.0.2"))
	dnsmasqCommand = append(dnsmasqCommand, "--dhcp-relay", fmt.Sprintf("%s,%s", tor.nodeAddress.IP.String(), "10.71.0.3"))
	dnsmasqCommand = append(dnsmasqCommand, "--dhcp-relay", fmt.Sprintf("%s,%s", tor.nodeAddress.IP.String(), "10.71.0.4"))
	dnsmasqCommand = append(dnsmasqCommand, "--dhcp-relay", fmt.Sprintf("%s,%s", tor.nodeAddress.IP.String(), "10.71.0.5"))
	dnsmasqCommand = append(dnsmasqCommand, "--dhcp-relay", fmt.Sprintf("%s,%s", tor.nodeAddress.IP.String(), "10.71.0.6"))
	dnsmasqCommand = append(dnsmasqCommand, "--dhcp-relay", fmt.Sprintf("%s,%s", tor.nodeAddress.IP.String(), "10.71.0.7"))

	torNs.Apps = append(torNs.Apps, &types.NetNSAppSpec{
		Name:    "dnsmasq",
		Command: dnsmasqCommand,
	})

	return torNs
}
