neco-worker
===========

`neco-worker` is a daemon program for handle update process.

Usage
-----

```console
$ neco-worker [OPTIONS]
```

Option    | Default value               | Description
------    | -------------               | -----------
`-config` | `/etc/neco/neco-worker.yml` | Configuration file path.

### Configuration file

Configuration file is YAML format.

Name         | Type            | Default | Description
----         | ----            | ------- | -----------
`http_proxy` | string          | -       | http proxy URL for internet connection. It is ignored for etcd and sabakan connections.
`etcd`       | etcdutil.Config | -       | etcd configuration defined in [etcdutil][]

Bootstrapping
-------------

When `neco-worker` is started at the first time, it compares its debian package
version against the current system version recorded in [etcd](etcd.md).

If the version matches, it installs programs as specified in `artifacts.go`.
If not match, `neco-worker` updates itself with the new debian package and retry.

Updating programs
-----------------

`neco-worker` watches [etcd](etcd.md) for events from `neco-updater`.
When there is a new version of neco, it updates itself by installing
the new debian package, then start automatic update process.

[etcdutil]: https://github.com/cybozu-go/etcdutil
