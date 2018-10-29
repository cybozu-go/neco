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

![architecture](http://www.plantuml.com/plantuml/png/ZPInZjim38PtFGNXAG5YFq26uj0Rsg50Xrkxo1BZ2d4aLvBlUZZatIl2jAG4QIwRHNxyHT8KdqAKFiwdKNXKKTfXHB2e77m8W4rocODKCNI3su8Ca0t9MmAQCFVgfFTWR98RnntCavOHs_JTK1ZRqTyEOph8NXBEPqzdSQuI6z2Y9pAdGJHRdOSFfaiPBKiHH-VrIAHGevirDDzC_3xt3I63YR_ddce7wpHtWaChtqKHvEiq9W46qtTYpgi6HgKd6SAR9g2S_gTNYAnQJAlsUKt-popVE-C80_eclLfDEHkbiUW3RAYVHsbte8wuWu3-q7NTTlb19pbWABBFpkFF5tXUe-67iVDVGa4bbmjNzomyDPLgXkQhSn5UqB-Yfo3ewVnmQZcjWbo6fZR09DMHaePDQKyLEXq72j8o9kc0m3UGIrCFDrkW3b3APO1QxTvi-wMC-JxFdA3kPY27xC5Zz0PV4LAjmJWh-CS-Wd8l7q55VaEGXhhEaU03-amMa7LB6nEhSHhT-ms86fRTopnaNwRtG9RHIIqkXl8koSDq3n7La_-ilWhDchgdBK9Ia5B267QGRkGgfDLW1cjYYWw2_Zgq7EDnC269BK_Ls8ExBhswxMP9To31so1ZrGQgiCfAfJEtRaLOzmlyik1dkooSUi6A9xGwRV1_)
<!-- go to http://www.plantuml.com/plantuml/ and enter the above URL to edit the diagram. -->

Repository contents
-------------------

### neco

`neco` is a deploy automation tool written in Go.
It installs and updates miscellaneous utilities in boot servers.

### neco-updater

`neco-updater` is a background service to detect new releases of `neco` and
invokes `neco update-all` to automate maintenance of boot servers.

Neco itself is released as a Debian package in [releases][].

### Test suite

[dctest](dctest/) directory contains test suites to run integration
tests in a virtual data center environment.

[Placemat][placemat] and [placemat-menu], products from Neco, are tools
to create complex network topology and virtual servers using Linux
networking stacks, [rkt][], and [QEMU][].

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
