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

`/usr/share/neco/ignitions/roles/ROLE/site.yml`
-----------------------------------------------

Ignitions registered by neco-worker to sabakan are included in `/usr/share/neco/ignitions/roles` for each roles.
neco-worker identify role and its ignitions by listing the directory.
The directory must contain an entry point file `site.yml`, which defined in [sabakan ignition spec.][ignition.md].


[etcdutil]: https://github.com/cybozu-go/etcdutil
[ignition.md]: https://github.com/cybozu-go/sabakan/blob/master/docs/ignition.md
