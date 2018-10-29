[![GitHub release](https://img.shields.io/github/release/cybozu-go/neco.svg?maxAge=60)][releases]
[![CircleCI](https://circleci.com/gh/cybozu-go/neco.svg?style=svg)](https://circleci.com/gh/cybozu-go/neco)
[![GoDoc](https://godoc.org/github.com/cybozu-go/neco?status.svg)][godoc]
[![Go Report Card](https://goreportcard.com/badge/github.com/cybozu-go/neco)](https://goreportcard.com/report/github.com/cybozu-go/neco)

Neco
====

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

To share and keep data between netboot servers, [etcd][] is used widely.
To protect data and authorize users, [Vault][] is used too.

As to network, Neco uses layer-3 technologies only.  Specifically,
[BGP][] is used to advertise/receive routes together with [BFD][] to
reduce route convergence and [ECMP][] for high availability.

(TBW: diagram)

Repository contents
-------------------

### neco

`neco` is a deploy automation tool written in Go.
It installs and updates miscellaneous utilities in boot servers.

See [docs/neco.md](docs/neco.md)

### neco-updater

`neco-updater` is a background service to detect new releases of `neco` and
invokes `neco update-all` to automate maintenance of boot servers.

Neco itself is released as a Debian package in [releases][].

See [docs/neco-updater.md](docs/neco-updater.md)

### generate-artifacts

`generate-artifacts` is a command-line tool to generate `artifacts.go` which is a collection of latest components.

See [docs/generate-artifacts.md](docs/generate-artifacts.md)

### Test suite

[dctest](dctest/) directory contains test suites to run integration
tests in a virtual data center environment.

[Placemat][placemat] and [placemat-menu], products from Neco, are tools
to create complex network topology and virtual servers using Linux
networking stacks, [rkt][], and [QEMU][].

### CI/CD

CI/CD in **Neco** is running by CircleCI. It will release neco deb package after passing all requirements. 
Then `neco-updater` service is updated automatically.

See [docs/cicd.md](docs/cicd.md)

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
[placemat-menu]: https://github.com/cybozu-go/placemat-menu
[rkt]: https://coreos.com/rkt/
[QEMU]: https://www.qemu.org/
