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
etcd connection refused.

It also periodically checks GitHub release of this repository.
To prevent rate limits for GitHub, it is highly recommended that
set personal access token by `neco config set github-token TOKEN`.

There are configurations for `neco-updater` to limit the time of release check.
The configuration is stored in etcd.

We can set the configuration by `neco config set release-time "cron1" "cron2"...` and `neco config set release-timezone "Asia/Tokyo"`.
It accepts multiple cron expressions and it's evaluated same as [neco-rebooter](./neco-rebooter.md).

The default value of `release-time` is `* * * * *` and `release-timezone` is `Asia/Tokyo`.
