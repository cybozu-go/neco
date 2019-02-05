neco-updater
============

`neco-updater` is a daemon program for handle update condition of `neco-worker`.

Usage
-----

```console
$ neco-updater [OPTIONS]
```

Option     | Default value          | Description
------     | -------------          | -----------
`--config` | `/etc/neco/config.yml` | Configuration file path.

`neco-updater` will notify status to webhook URL when update
process is completed or stopped. This URL keeps on memory to prevent
etcd connection refused. To prevent rate limits for GitHub, It recommends
to set personal access token by `neco config set github-token TOKEN`.

