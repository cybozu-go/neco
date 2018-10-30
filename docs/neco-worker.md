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
Parameters are defined by [cybozu-go/etcdutil](https://github.com/cybozu-go/etcdutil), and not shown below will use default values of the etcdutil.

Name         | Type   | Default | Description
----         | ----   | ------- | -----------
`http_proxy` | string | -       | http proxy URL for internet connection. It is ignored for etcd and sabakan connections.
