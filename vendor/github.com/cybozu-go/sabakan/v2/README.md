[![GitHub release](https://img.shields.io/github/release/cybozu-go/sabakan.svg?maxAge=60)][releases]
[![CircleCI](https://circleci.com/gh/cybozu-go/sabakan.svg?style=svg)](https://circleci.com/gh/cybozu-go/sabakan)
[![GoDoc](https://godoc.org/github.com/cybozu-go/sabakan?status.svg)][godoc]
[![Go Report Card](https://goreportcard.com/badge/github.com/cybozu-go/sabakan)](https://goreportcard.com/report/github.com/cybozu-go/sabakan)

Sabakan
=======

![sabakan architecture](http://www.plantuml.com/plantuml/svg/ZOv1ImCn58Jl-HL32zv2wUvDH4eLseE7Wlq3hvlNMimcahnKHFplIcAqIrtmai1yysPc4OM2fDwgm9sGErZ6XAKpw6oAmc62TmKO4jfHP6H4CV_pCT2CWLO12kKOslXNfyi1fgj0Rt-XTe0QQ6rvBte8FzJv_ANtWaSEfxh-bqNQqJCvJ1-EXoTPsiJgf_Ic95TFRHpHsmlzQuJpXf4VYlcVNqgD6ixfn7xFMGLcfwfMqczh_3N8c1cReym2z_vKGbKkWKulvyxxzTq6LrXljnliVS3EUydEvb_E1JkJUli9)
<!-- go to http://www.plantuml.com/plantuml/ and enter the above URL to edit the diagram. -->

Sabakan is a versatile network boot server designed for large on-premise data centers.
Currently, it is made only for [CoreOS Container Linux](https://coreos.com/os/docs/latest/).

**Project Status**: GA (General Availability)

Features
--------

* High availability

    High availability of sabakan is just as easy as running multiple sabakan servers.

    Sabakan data are stored and shared in [etcd][].  For example, DHCP lease information
    are shared between sabakan instances to avoid conflicts.

* Machine inventory with IPAM (IP address management)

    Sabakan keeps an inventory of machines in a data center.  Their IP addresses
    are automatically assigned by sabakan.

* DHCP service

    Sabakan provides DHCP service that supports [UEFI HTTP Boot][HTTPBoot]
    and [iPXE][] HTTP Boot.  It also supports DHCP relay request to make DHCP service
    highly available.

* HTTP service (network file server)

    Sabakan provides HTTP service for network boot clients.  Users can upload
    any kind of files other than OS images to sabakan.  Clients can download them
    to initialize the system after boot.

* Template system for Ignition

    [Ignition][] is a boot provisioning system for CoreOS Container Linux.
    Ignition configuration is not friendly for operators as it is written in a plain JSON.

    Sabakan provides a friendly and super versatile template system for Ignition configurations.
    For each client machine, sabakan renders Ignition configuration from templates.

* Life-cycle management

    Machines in the inventory has a life-cycle status.  The status can be changed through
    REST API.  Users can build an automatic status controller to mark machines as unhealthy,
    unreachable, retiring, or retired.

* Disk encryption support

    To help implementing full disk encryption on client machines, sabakan accepts and stores
    encrypted disk encryption keys.  The key can be downloaded in the next boot to decrypt
    disks.
    
    `sabakan-cryptsetup` is a tool for clients to encrypt disks; the tool generates a disk
    encryption key, encrypts it, and sends the encrypted key to sabakan.  In the next boot,
    it downloads the encrypted key from sabakan, decrypts it, then uses it to decrypt disks.

* Audit logs

    To track problems and life-cycle events, sabakan keeps operation logs
    within its etcd storage.

Programs
--------

This repository contains these programs:

* `sabakan`: the network service to manage servers.
* `sabactl`: CLI tool for `sabakan`.
* `sabakan-cryptsetup`: a utility to encrypt a block device using [dm-crypt][].

To see their usage, run them with `-h` option.

Documentation
-------------

[docs](docs/) directory contains tutorials and specifications.

Read [getting started](docs/getting_started.md) first.

Examples
--------

[mtest/](./mtest/) directory contains a set of utilities to setup sabakan on Ubuntu virtual machines.

[testadata/](./testdata/) directory contains a sample Ignition template.

An example of production usage can be found in [github.com/cybozu-go/neco](https://github.com/cybozu-go/neco).
The repository bootstraps a full data center system using etcd, vault, sabakan, and many other tools.

Usage
-----

Run sabakan with docker

```console
# create directory to store OS images
$ sudo mkdir -p /var/lib/sabakan

# -advertise-url is the canonical URL of this sabakan.
$ docker run -d --read-only --cap-drop ALL --cap-add NET_BIND_SERVICE \
    --network host --name sabakan \
    --mount type=bind,source=/var/lib/sabakan,target=/var/lib/sabakan \
    quay.io/cybozu/sabakan:2.2 \
    -etcd-endpoints http://foo.bar:2379,http://zot.bar:2379 \
    -advertise-url http://12.34.56.78:10080
```

License
-------

Sabakan is licensed under MIT license.

[releases]: https://github.com/cybozu-go/sabakan/releases
[godoc]: https://godoc.org/github.com/cybozu-go/sabakan
[etcd]: https://coreos.com/etcd/
[HTTPBoot]: https://github.com/tianocore/tianocore.github.io/wiki/HTTP-Boot
[iPXE]: https://ipxe.org/
[Ignition]: https://coreos.com/ignition/
[dm-crypt]: https://gitlab.com/cryptsetup/cryptsetup/wikis/DMCrypt
