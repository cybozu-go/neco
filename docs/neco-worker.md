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

[etcdutil]: https://github.com/cybozu-go/etcdutil
