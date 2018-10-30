neco-updater
============

`neco-updater` is a daemon program for handle update condition of `neco-worker`.

Usage
-----

```console
$ neco-updater [OPTIONS]
```

Option    | Default value                | Description
------    | -------------                | -----------
`-config` | `/etc/neco/neco-updater.yml` | Configuration file path.

`neco-updater` will notify status to webhook URL when update
process is completed or stopped. This URL keeps on memory to prevent
etcd connection refused.

### Configuration file

Configuration file is YAML format.
Parameters are defined by [cybozu-go/etcdutil](https://github.com/cybozu-go/etcdutil), and not shown below will use default values of the etcdutil.

Name         | Type   | Default | Description
----         | ----   | ------- | -----------
`http_proxy` | string | -       | http proxy URL for internet connection. It is ignored for etcd and sabakan connections.
