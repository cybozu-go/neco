[![GitHub release](https://img.shields.io/github/release/cybozu-go/neco.svg?maxAge=60)][releases]
[![CircleCI](https://circleci.com/gh/cybozu-go/neco.svg?style=svg)](https://circleci.com/gh/cybozu-go/neco)
[![GoDoc](https://godoc.org/github.com/cybozu-go/neco?status.svg)][godoc]
[![Go Report Card](https://goreportcard.com/badge/github.com/cybozu-go/neco)](https://goreportcard.com/report/github.com/cybozu-go/neco)

<img src="./neco_logo.svg" width="100" alt="neco logo" />

**Neco** is a collection of tools to build and maintain large data center
infrastructure.  It is used in [cybozu.com](https://www.cybozu.com/), the
top B2B groupware service in Japan.

**Project Status**: Perpetual beta.

Architecture
------------

Neco adopts [network booting][netboot] to manage thousands of servers
efficiently.  The center of Neco is therefore a few netboot servers.

[neco-ubuntu][] creates a custom Ubuntu installer to setup netboot servers.

[Sabakan][sabakan], a product from Neco, implements functions necessary
for netboot including [DHCP][], [UEFI HTTP Boot][], [iPXE][], [ignition][]
and file server service to download assets.

![sabakan_architecture](http://www.plantuml.com/plantuml/svg/ZOv1IyGm58Jl-HN3BdZBOTkRY2ohu1uyBBX_uBLvhMAQIFAYYFZVBIbHIoruoOFvPZApZq91qc1Lu5R8zPQnOMaDMfkYSDZWGm66X1gAZ8mevhjR0zKQg4UWC8MXZNzpUWfWUnVe_IzKpr05hIrtekVmK_sUV_1UyC3XjQp_OP4QUYQ7xVrJ_oW7crXzbrvDFnTFQLpHwuK-Zd3UCF93CT_TKgfK1j3fHL-Ny2LkZpSdNE1uFf_G-O36Ur7P_o_ddfr9W_q2)
<!-- go to http://www.plantuml.com/plantuml/ and enter the above URL to edit the diagram. -->

To share and keep data between netboot servers, [etcd][] is used widely.
To protect data and authorize users, [Vault][] is used too.

![etcd_vault_architecture](http://www.plantuml.com/plantuml/svg/TOwzQWCn48HxFSLmgKM8hmiXc3HvW90kpKPQcmDPEdPN3YRatUEpyC-1hv5WFhwTMQkHMDqb9noCyZOnEhOG4L9LO-dmwu18Hj-aZ1CYFVrFIs2r1FeZS6WoV2m_sJS13-z2Xtkedw4Ll4-yCJ-7V-vs_bifXW-M_NdzbUsf9fiaFhXBsqixsU2bw5xQpzEfbu8LGVUfB2W26iSq1BAXv0wagCVSJS_PV6tgCyPgZrisA0TXqwyyg5P6OB5XCvrWdOjjxLMCPEJMd6FTfNy0)
<!-- go to http://www.plantuml.com/plantuml/ and enter the above URL to edit the diagram. -->

As to network, Neco uses layer-3 technologies only.  Specifically,
[BGP][] is used to advertise/receive routes together with [BFD][] to
reduce route convergence and [ECMP][] for high availability.

![network_architecture](http://www.plantuml.com/plantuml/svg/ZLB1Ri8m3BtdAopk7E8ZcYO6aoPkKxexYjeCAjgu2kdGDF7l2mviHR9eBr2_z_py77bvZ3R4lctKyL3xpWRRGbDx5xyx1nJYdbJPK5_1naSN4gw2AwFrkyR1R4t1GN6gOxcVWJq2rp_gFDGKNN8RYXZGqsJ8ijjeU9fNTFBpPnwaUDeVb6qb45Nc_c5ZD2p0hTxUCuKI9NIXt7L7gSwM1xjBAxqKinGVm5ELAfDWdG60mU8VPAvhW-R54w2px5veg8yEZFji4aQ1jMOWFIl-VU2FDt-SxezZ_YkY28KBNowtS2tOhpRcDGlIn_QYaltMr7QN8DaIT3wiGdoIZYgcqx_UwX4UHpBfINdmcWT7yk187XpDWuCynWpkaCF20kfqRZAB3rb-_7i5okuoYp8hkQlB0cUrTBxgs-ON)
<!-- go to http://www.plantuml.com/plantuml/ and enter the above URL to edit the diagram. -->

Repository contents
-------------------

### neco

`neco` is a deploy automation tool written in Go.
It installs and updates miscellaneous utilities in boot servers.

See [docs/neco.md](docs/neco.md)

### neco-updater

`neco-updater` is a background service to detect new releases of `neco`.
When detected, it sends the update information to `neco-worker` through etcd.

Neco itself is released as a Debian package in [releases][].

See [docs/neco-updater.md](docs/neco-updater.md)

### neco-worker

`neco-worker` is an automate maintenance service. It installs/updates `neco` package, applications,
and sabakan contents when receives information from `neco-updater` through etcd.

See [docs/neco-worker.md](docs/neco-worker.md)

### generate-artifacts

`generate-artifacts` is a command-line tool to generate `artifacts.go` which is a collection of latest components.

See [docs/generate-artifacts.md](docs/generate-artifacts.md)

### Test suite

[dctest](dctest/) directory contains test suites to run integration
tests in a virtual data center environment.

[Placemat][placemat] is a tool to create arbitrarily complex network topology
and virtual servers using Linux networking stacks, [rkt][], and [QEMU][].

See [docs/dctest.md](docs/dctest.md)

### CI/CD

CI/CD in **Neco** is running by CircleCI.
Then `neco-updater` service is updated automatically.

See [docs/cicd.md](docs/cicd.md)

### Run unit tests at local machine

First, start up etcd server manually.

```console
$ make start-etcd
```

Then, run `go test` on another terminal.

```console
$ go test -v -count=1 -race -mod=vendor ./...
```

### GCP suite

[gcp](gcp/) directory contains utilities for provisioning the Google Compute Engine services for Neco project.
Test suite, and other Neco github projects deploy GCE instance based on an image created by this tool.

See [docs/gcp](docs/gcp/)

### `git-neco`

`git-neco` is a git extension utility to help Neco developers.

Documentation
-------------

[docs](docs/) directory contains documents about designs and specifications.

[releases]: https://github.com/cybozu-go/neco/releases
[godoc]: https://godoc.org/github.com/cybozu-go/neco
[netboot]: https://en.wikipedia.org/wiki/Network_booting
[neco-ubuntu]: https://github.com/cybozu/neco-ubuntu
[sabakan]: https://github.com/cybozu-go/sabakan
[DHCP]: https://en.wikipedia.org/wiki/Dynamic_Host_Configuration_Protocol
[UEFI HTTP Boot]: https://github.com/tianocore/tianocore.github.io/wiki/HTTP-Boot
[iPXE]: https://ipxe.org/
[ignition]: https://github.com/coreos/ignition
[etcd]: http://etcd.io/
[Vault]: http://vaultproject.io/
[BGP]: https://en.wikipedia.org/wiki/Border_Gateway_Protocol
[BFD]: https://en.wikipedia.org/wiki/Bidirectional_Forwarding_Detection
[ECMP]: https://en.wikipedia.org/wiki/Equal-cost_multi-path_routing
[placemat]: https://github.com/cybozu-go/placemat
[rkt]: https://coreos.com/rkt/
[QEMU]: https://www.qemu.org/
