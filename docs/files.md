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

[etcdutil]: https://github.com/cybozu-go/etcdutil
