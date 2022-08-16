Files and directories
=====================

`/etc/neco`
-----------

This directory holds miscellaneous configuration files for Neco.

### `rack`

Stores the rack number of the boot server.

### `cluster`

Stores the cluster ID where the boot server belongs.

### `server.crt` and `server.key`

TLS server certificates for this boot server.

### `etcd.crt` and `etcd.key`

TLS client certificates for etcd authentication for Neco tools.

### `config.yml`

etcd configuration defined in [etcdutil][]

### `sabakan_ipam.json`

IPAM configuration for [sabakan][].

`/usr/share/neco`
-----------------

This directory holds miscellaneous configuration files for applications installed by Neco.

### `dhcp.json`

[sabakan][] configuration.

### `cke-template.yml`

[cke][] configuration.

### `coil-deploy.yml` and `coil-rbac.yml`

Manifests for [coil][].

### `squid.yml`

Manifest for [squid][].

### `unbound.yml`

Manifest for [unbound][].

`/usr/share/neco/ignitions/roles/ROLE/<site.yml|site-<cluster>.yml>`
-----------------------------------------------

Ignitions registered by neco-worker to sabakan are included in `/usr/share/neco/ignitions/roles` for each roles.
neco-worker identify role and its ignitions by listing the directory.
The directory must contain an entry point file `site.yml|site-<cluster>.yml`, which is defined in [sabakan ignition spec.][ignition.md].
A file named `site-<cluster>.yml` is used to place configuration files for each environment.
`<cluster>` is the name of the cluster and defined `/etc/neco/cluster`.
If `site-<cluster>.yml` does not exist, `site.yml` is used.

`/etc/udev` (only for ss)
-------------------------

### `crypt-base-path`

The shell script to determine link names located under `/dev/crypt-disk/by-path` based on the bus where the devices are connected. This script is used in udev rule `99-neco.rules`.

### `rules.d/99-neco.rules`

The udev rules to create the links located under `/dev/crypt-disk/by-path`.


[etcdutil]: https://github.com/cybozu-go/etcdutil
[sabakan]: https://github.com/cybozu-go/sabakan
[cke]: https://github.com/cybozu-go/cke
[coil]: https://github.com/cybozu-go/coil
[squid]: http://www.squid-cache.org/
[unbound]: https://nlnetlabs.nl/projects/unbound/about/
[ignition.md]: https://github.com/cybozu-go/sabakan/blob/master/docs/ignition.md
