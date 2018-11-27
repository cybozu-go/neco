[![GitHub release](https://img.shields.io/github/release/cybozu-go/sabakan.svg?maxAge=60)][releases]
[![CircleCI](https://circleci.com/gh/cybozu-go/sabakan.svg?style=svg)](https://circleci.com/gh/cybozu-go/sabakan)
[![GoDoc](https://godoc.org/github.com/cybozu-go/sabakan?status.svg)][godoc]
[![Go Report Card](https://goreportcard.com/badge/github.com/cybozu-go/sabakan)](https://goreportcard.com/report/github.com/cybozu-go/sabakan)

Sabakan
=======

![sabakan architecture](http://www.plantuml.com/plantuml/svg/ZOv1ImCn58Jl-HL32zv2wUvDH4eLseE7Wlq3hvlNMimcahnKHFplIcAqIrtmai1yysPc4OM2fDwgm9sGErZ6XAKpw6oAmc62TmKO4jfHP6H4CV_pCT2CWLO12kKOslXNfyi1fgj0Rt-XTe0QQ6rvBte8FzJv_ANtWaSEfxh-bqNQqJCvJ1-EXoTPsiJgf_Ic95TFRHpHsmlzQuJpXf4VYlcVNqgD6ixfn7xFMGLcfwfMqczh_3N8c1cReym2z_vKGbKkWKulvyxxzTq6LrXljnliVS3EUydEvb_E1JkJUli9)
<!-- go to http://www.plantuml.com/plantuml/ and enter the above URL to edit the diagram. -->

Sabakan is an integration service to automate bare-metal server management.
It uses [etcd][] as a backend datastore for strong consistency and high availability.

**Project Status**: Initial development.

Features
--------

* High availability

    Thanks to etcd, sabakan can run multiple instances while maintaining
    strong consistency.  For instance, DHCP lease information are shared
    among sabakan instances to avoid conflicts.

* Machine inventory / IP address management (IPAM)

    Machines in a data center can be registered with sabakan's inventory.
    In addition, sabakan assigns IP addresses automatically to machines.

* DHCP service

    Sabakan provides DHCP service that supports [UEFI HTTP Boot][HTTPBoot]
    and [iPXE][] HTTP Boot.

* HTTP service

    Sabakan serves OS images to machines via HTTP.

* Distributed asset management

    In order to help initialization of client servers, sabakan can work
    as a file server from which clients can download assets via HTTP.
    Assets are automatically synchronized between sabakan servers.

* Encryption key store

    Sabakan provides REST API to store and retrieve encryption keys
    to help automated disk encryption/decryption.

* Life-cycle management

    Sabakan provides API to change server status for [life-cycle management](docs/lifecycle.md).

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

License
-------

Sabakan is licensed under MIT license.

[releases]: https://github.com/cybozu-go/sabakan/releases
[godoc]: https://godoc.org/github.com/cybozu-go/sabakan
[etcd]: https://coreos.com/etcd/
[HTTPBoot]: https://github.com/tianocore/tianocore.github.io/wiki/HTTP-Boot
[iPXE]: https://ipxe.org/
[dm-crypt]: https://gitlab.com/cryptsetup/cryptsetup/wikis/DMCrypt
