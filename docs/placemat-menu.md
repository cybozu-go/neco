# placemat-menu

`placemat-menu` reads a tiny YAML configuration file and generates a large set of configuration
files for [placemat](https://github.com/cybozu-go/placemat) to construct a virtual data center.

## Network Design

The cluster of the overview is in the following figure.

![Network Design](http://www.plantuml.com/plantuml/png/hPF1IiGm48RlUOgX9pq4d3p1uYBePGNhJKH26tVBran9P-a-lcsIjIarwyLBwNp__y_03Ddqh1sVlbhXJCNQxbi3nIFr33TFbespXcyBq3qqiH9LIwSQYeVpLEiMHZQGEtgJEV_epvrncXkoi4iCr74wQ4lEG3aqN1syN8rrgfTTOnU6VWBujqMbbbTwyOhJrV7kWydXLJMRnQjP35bXgJQX6Ro9UoA6tKY4b59iI_-FQQ5yKQPAUL7Uasxu7zqkHmHPqs1bsFVqYI3kTurKHAxP7rY6EtlGca-M_gmX6bF9hdE2-aN0N0AJXChDM0gv1EOISSRSTT5kvchDSUN7cQibtnXRJ-_j6m00)

The network is designed as a spine-and-leaf architecture.  There is a *core
switches* at the top of the cluster.  The *spine switches* connects core
switches and racks.  It is a L3 switch to routing between inter cluster,
Internet, and other networks.

The *operation network* is a network for management of the cluster, it is
normally used for administrators or SRE of the cluster.  Users can reach only
boot servers from the operation network.  The *external network* is other
cluster or service.  It can reach to the cluster via exposed Ingress IP
addresses provided by the cluster.

Each rack is separated as an individual L3 network.  Although, nodes in a rack
can connect as L2 networks, the nodes over racks are able to each nodes via L3
network.  The nodes uses BGP routing to connect each nodes over all clusters.
The node has a virtual (dummy) network to connect inter-rack nodes.  Its
addresses are advertised to switches and nodes by BGP.  The address of the
physical interface are scoped as link-local, they are used for only L2 network.

The rack has two top of rack (ToR) switches to load balancing network traffics
and increase reliability.  Every node in the rack have two network interfaces
named *node1* and *node2* network.  The node1 interface connect to one ToR
switch, and node2 connects to other ToR switch, respectively.  The advertised
network address in the cluster via BGP is called *node0*.  Additionally, the
boot node has *bastion* network interface.  It is also virtual network, which
is advertised to the operation network.  So uses reach the boot servers via
this IP addresses from operation network.

### Communication between the Internet

The core switch does not accept incoming connection from the Internet.

The core switch runs squid HTTP proxy on port 3128 for egress communication.

## Usage

    $ placemat-menu -f <source.yml> [-o <output dir>]

## Example

[menu/example](../menu/example) has an example `menu.yml`.

## YAML Specification

The source YAML of the `placemat-menu` consists of the set of the following resources:

* Network
* Inventory
* Image
* Node

### Network resource

Network resource defines IP offsets and ranges to assign each nodes and switches

```yaml
kind: Network
spec:
  ipam-config: ipam.json
  asn-base: 64600
  internet: 10.0.0.0/24
  spine-tor: 10.0.1.0
  core-spine: 10.0.2.0/31
  core-external: 10.0.3.0/24
  core-operation: 10.0.4.0/24
  proxy: 10.0.49.3
  pod: 10.64.0.0/14
  exposed:
    bastion: 10.72.48.0/26
    loadbalancer: 10.72.32.0/20
    ingress: 10.72.48.64/26
    global: 172.19.0.0/26
```

- `ipam-config`: The path of configuration file of IP address assignment.
The details of this file are described in the [Sabakan spec](https://github.com/cybozu-go/sabakan/blob/master/docs/ipam.md#ipamconfig).
For `placemat-menu`, `node-ip-per-node` must be 3 and `node-index-offset` must be 3.
The node address and ToR address are assigned based on this file's content.
The following example is assigned addresses when `"node-ipv4-pool": "10.69.0.0/20"`,
`"node-ipv4-range-size": 6`, and `"node-ipv4-range-mask": 26` are specified.
    - rack0 node0 network: 10.69.0.0/26      # node + 64 * 0
    - rack0 node1 network: 10.69.0.64/26     # node + 64 * 1
    - rack0 node2 network: 10.69.0.128/26    # node + 64 * 2<br><br>
    - rack0-tor1 eth1: 10.69.0.65/26         # rack0 node1 network + 1
    - rack0-tor2 eth1: 10.69.0.129/26        # rack0 node2 network + 1<br><br>
    - boot-0 node0: 10.69.0.3/32             # rack0 node0 network + 3
    - boot-0 node1(eth0): 10.69.0.67/26      # rack0 node1 network + 3
    - boot-0 node2(eth1): 10.69.0.131/26     # rack0 node2 network + 3<br><br>
    - rack0-cs1 node0: 10.69.0.4/32          # rack0 node0 network + 4
    - rack0-cs1 node1(eth0): 10.69.0.68/26   # rack0 node1 network + 4
    - rack0-cs1 node2(eth1): 10.69.0.132/26  # rack0 node2 network + 4<br><br>
    - rack0-cs2 node0: 10.69.0.5/32          # rack0 node0 network + 5
    - rack0-cs2 node1(eth0): 10.69.0.69/26   # rack0 node1 network + 5
    - rack0-cs2 node2(eth1): 10.69.0.133/26  # rack0 node2 network + 5<br><br>
    - rack1 node0 network: 10.69.0.192/26    # node + 64 * 3
    - rack1 node1 network: 10.69.1.0/26      # node + 64 * 4
    - rack1 node2 network: 10.69.1.64/26     # node + 64 * 5<br><br>
    - rack1-tor1 eth1: 10.69.1.1/26          # rack1 node1 network + 1
    - rack1-tor2 eth1: 10.69.1.65/26         # rack1 node2 network + 1<br><br>
    - boot-1 node0: 10.69.0.195/32           # rack1 node0 network + 3
    - boot-1 node1(eth0): 10.69.1.3/26       # rack1 node1 network + 3
    - boot-1 node2(eth1): 10.69.1.67/26      # rack1 node2 network + 3<br><br>
    - rack1-cs1 node0: 10.69.0.196/32        # rack1 node0 network + 4
    - rack1-cs1 node1(eth0): 10.69.1.4/26    # rack1 node1 network + 4
    - rack1-cs1 node2(eth1): 10.69.1.68/26   # rack1 node2 network + 4<br><br>
    - rack1-cs2 node0: 10.69.0.197/32        # rack1 node0 network + 5
    - rack1-cs2 node1(eth0): 10.69.1.5/26    # rack1 node1 network + 5
    - rack1-cs2 node2(eth1): 10.69.1.69/26   # rack1 node2 network + 5<br><br>

- `asn-base`:  The offset of the private AS number (ASN) assigned for each BGP
routers.  The ASN of the ext-vm is set as `asn-base - 2`, and the spine
switches are set as `asn-base - 1`.  The following example is ANS assignments
for each switches when `64000` is specified:

    - ext-vm: 64598
    - spine: 64599
    - rack0: 64600
    - rack1: 64601
    - rack2: 64602

- `internet`: The network address assigned for the internet network.  The
addresses in this network are assigned for each links between ext-vm and
spine switches.  The following example is IP addresses assigned for each
switches when `10.0.0.0/24` is specified:

    - host-vm: 10.0.0.1
    - ext-vm: 10.0.0.2
    - spine0: 10.0.0.3
    - spine1: 10.0.0.4
    - spine2: 10.0.0.5

- `spine-tor`: The offset address assigned each switches between spine switched
and ToR switches.  The length of the prefix is `/31`.  Four addresses are
assigned for a rack, since each rack has two ToR switches.  The following
example is assigned addresses when `10.0.1.0` is specified:

    - spine0-to-rack0-tor1: 10.0.1.0/31
    - rack0-tor1-to-spine0: 10.0.1.1/31<br><br>
    - spine0-to-rack0-tor2: 10.0.1.2/31
    - rack0-tor2-to-spine0: 10.0.1.3/31<br><br>
    - spine0-to-rack1-tor1: 10.0.1.4/31
    - rack1-tor1-to-spine0: 10.0.1.5/31<br><br>
    - spine0-to-rack1-tor2: 10.0.1.6/31
    - rack1-tor2-to-spine0: 10.0.1.7/31

- `core-spine` The network address between the core switch and spines switches.

- `core-external`: The network address between the core and the external network.

- `core-operation`: The network address between the core switch and the operation network.

- `proxy`: The IP address of the HTTP/HTTPS proxy server to
    the Internet running on the core switch.

- `ntp`: The IP addresses of the NTP servers running on the core switch.

- `pod`: The network address advertised to outside of the cluster. Different from `exposed`, `pod` network will not accept connections from outside.

- `exposed`: The network addresses advertise to outside of the cluster
    - `bastion`: The bastion network addresses, whey are also advertised to the
        external of the cluster.  They are assigned for the boot servers, and they able
        to be accessed from the internet network.  The following example is the
        assigned addresses when `10.72.48.0/26` is set:
        - boot-0: 10.72.48.0/32
        - boot-1: 10.72.48.1/32
        - boot-2: 10.72.48.2/32
    - `loadbalancer`: The network addresses used for the load balancer exposed to the external address.
    - `ingress`: The ingress network addresses from the external address.
    - `global`: The global network addresses to reach Internet.

### Inventory resource

Inventory resource presents the specifications of the nodes excluding boot
servers.  This resource contains the number of the computation server (cs) and
storage server (ss) in the rack.  The rack must have a boot server, so the
resource does not contain a configuration for the boot server.

```yaml
kind: Inventory
spec:
  spine: 3
  rack:
    - cs: 2
      ss: 1
    - cs: 2
      ss: 1
    - cs: 2
      ss: 1
```

The above example, the cluster contains three spine switches and three racks.
Thus, there are six ToR switches, since each rack has two ToR switches.

The available properties are as following:

- `spine`: the number of the spine switches in the cluster.
- `rack`: the rack configurations
    - `cs`: the number of the computer servers (cs)
    - `ss`: the number of the storage servers (ss)

### Image resource

Image resource is the same as [Image resource of placemat](https://github.com/cybozu-go/placemat/blob/master/docs/resource.md#image-resource).

### Node resource

Node resource specify the resources of the machines.

```yaml
kind: Node
type: cs
spec:
  cpu: 2
  memory: 2G
  image: ubuntu-cloud-image
  data:
    - docker-lib-image
  uefi: true
  cloud-init-template: boot-seed.yml.template
  tpm: true
```

The available properties are as following:

- `type`: the machine type, following types are available
    - `boot`: boot servers
    - `cs`: computation servers
    - `ss`: storage servers
- `cpu`: The number of the virtual CPU cores
- `memory`: The size of the memory.
- `image`: The name of an image resource for boot (optional)
- `data`: The name of image resources for additional data (optional)
- `smbios`: The name of BIOS mode (optional. See [Node resource of placemat](https://github.com/cybozu-go/placemat/blob/master/docs/resource.md#node-resource))
- `uefi`: Use UEFI boot.
- `cloud-init-template`: The path of cloud-init template file.
- `tpm`: Use virtual TPM.

In a cloud-init template file, following attributes can be referenced.

- .Name: The node name
- .Rack: The rack information
    - Index: The logical number of rack

```yaml
#cloud-config
hostname: {{.Name}}
runcmd:
- ["/extras/setup/setup-neco-network", "{{.Rack.Index}}"]
```
