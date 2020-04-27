[![GitHub release](https://img.shields.io/github/release/cybozu-go/cke.svg?maxAge=60)][releases]
[![CircleCI](https://circleci.com/gh/cybozu-go/cke.svg?style=svg)](https://circleci.com/gh/cybozu-go/cke)
[![GoDoc](https://godoc.org/github.com/cybozu-go/cke?status.svg)][godoc]
[![Go Report Card](https://goreportcard.com/badge/github.com/cybozu-go/cke)](https://goreportcard.com/report/github.com/cybozu-go/cke)
[![Docker Repository on Quay](https://quay.io/repository/cybozu/cke/status "Docker Repository on Quay")](https://quay.io/repository/cybozu/cke)
[![CII Best Practices](https://bestpractices.coreinfrastructure.org/projects/3391/badge)](https://bestpractices.coreinfrastructure.org/projects/3391)

Cybozu Kubernetes Engine
========================

<a href="https://landscape.cncf.io/format=card-mode&grouping=category&organization=cybozu&selected=cybozu-kubernetes-engine"><img src="https://raw.githubusercontent.com/cncf/artwork/master/projects/kubernetes/certified-kubernetes/versionless/color/certified-kubernetes-color.svg?sanitize=true" align="right" width="120px" alt="Kubernetes certification logo"></a>

**CKE** (Cybozu Kubernetes Engine) is a distributed service that automates [Kubernetes][] cluster management.

**Project Status**: Testing for production.

Requirements
------------

### CKE requirements

* [etcd][]
* [Vault][]

### Node OS Requirements

* Docker: etcd data is stored in Docker volumes.
* A user who belongs to `docker` group
* SSH access for the user

Features
--------

* Bootstrapping and life-cycle management.

    CKE can bootstrap a Kubernetes and [etcd][] cluster from scratch.
    CKE can also add or remove nodes to/from the Kubernetes and etcd cluster.

* In-place and fast upgrade of Kubernetes

    A version of CKE corresponds strictly to a single version of Kubernetes.
    Therefore, upgrading CKE will upgrade the managed Kubernetes.

    Unlike [kubeadm][] or similar tools, CKE can automatically upgrade
    its managed Kubernetes without draining nodes.  The time taken for
    the upgrade is not proportional to the number of nodes, so it is
    very fast.

* Managed etcd cluster

    CKE manages an etcd cluster for Kubernetes.
    Other applications may also store their data in the same etcd cluster.
    Backups of etcd data are automatically taken by CKE.

    Details are described in [docs/etcd.md](docs/etcd.md).

* CRI runtimes

    In addition to Docker, CRI runtimes such as [containerd][] or [cri-o][]
    can be used to run Kubernetes Pods.

* Kubernetes features:

    * HA control plane.
    * [RBAC][] is enabled.
    * Ready for [PodSecurityPolicy][]
    * Ready for API [aggregation](https://kubernetes.io/docs/tasks/access-kubernetes-api/configure-aggregation-layer/).
    * `Secret` data are [encrypted at rest](https://kubernetes.io/docs/tasks/administer-cluster/encrypt-data/).
    * [CNI][] network plugins.
    * [CoreDNS][] add-on.
    * Node-local DNS cache services.
    * Nodes can be registered with [Taints][].
    * Preparation of [Scheduler extender](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/scheduling/scheduler_extender.md).

* User-defined resources:

    CKE automatically creates or updates Kubernetes API resources such as Deployments,
    Namespaces, or CronJobs that are defined by users.  This feature helps users to
    automate Kubernetes cluster maintenance.

* Sabakan integration

    CKE can be integrated with [sabakan][], a service that automates physical
    server management, to generate cluster configuration automatically.

    Sabakan is not a requirement; cluster configuration can be supplied
    externally by a YAML file.

* High availability

    CKE stores its configurations in [etcd][] to share them among multiple instances.
    [Etcd][etcd] is also used to elect a leader instance that exclusively controls
    the Kubernetes cluster.

* Operation logs

    To track problems and life-cycle events, CKE keeps operation logs in etcd.

Programs
--------

This repository contains these programs:

* `cke`: the service.
* `ckecli`: CLI tool for `cke`.

To see their usage, run them with `-h` option.

Getting started
---------------

A demonstration of CKE running on docker is available at [example](example/) directory.

Documentation
-------------

[docs](docs/) directory contains tutorials and specifications.

Usage
-----

### Run CKE with docker

```console
$ docker run -d --read-only \
    --network host --name cke \
    quay.io/cybozu/cke:1.17 [options...]
```

### Install `ckecli` to host file system

```console
$ docker run --rm -u root:root \
    --entrypoint /usr/local/cke/install-tools \
    --mount type=bind,src=DIR,target=/host \
    quay.io/cybozu/cke:1.17
```

Feedback
--------

Please report bugs / issues to [GitHub issues](https://github.com/cybozu-go/cke/issues).

Feel free to send your pull requests!

License
-------

CKE is licensed under MIT license.

[releases]: https://github.com/cybozu-go/cke/releases
[godoc]: https://godoc.org/github.com/cybozu-go/cke
[Kubernetes]: https://kubernetes.io/
[etcd]: https://github.com/etcd-io/etcd
[kubeadm]: https://kubernetes.io/docs/reference/setup-tools/kubeadm/kubeadm/
[containerd]: https://containerd.io/
[cri-o]: https://cri-o.io/
[Vault]: https://www.vaultproject.io
[RBAC]: https://kubernetes.io/docs/reference/access-authn-authz/rbac/
[PodSecurityPolicy]: https://kubernetes.io/docs/concepts/policy/pod-security-policy/
[CNI]: https://kubernetes.io/docs/concepts/extend-kubernetes/compute-storage-net/network-plugins/
[CoreDNS]: https://coredns.io/
[sabakan]: https://github.com/cybozu-go/sabakan
[Taints]: https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/
