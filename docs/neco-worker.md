neco-worker
===========

`neco-worker` is a daemon program for handle update process.

Usage
-----

```console
$ neco-worker [OPTIONS]
```

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

It also checks latest GitHub release of debian package such as `etcdpasswd` and `neco`.
To prevent GitHub rate limits, it is highly recommended that
set personal access token by `neco config set github-token TOKEN`.
