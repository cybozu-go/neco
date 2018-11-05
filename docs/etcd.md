etcd schema
===========

Legend:
* `<prefix>` = `/neco/`

## `<prefix>/leader/`

This prefix is used to elect a leader `neco-updater` who is responsible to invoke
update process to `neco-worker`.

## `<prefix>/bootservers/<LRN>`

This prefix is current available boot servers. Current available boot server is
registered as a key `LRN`.  The value is empty.

## `<prefix>/install/<LRN>/containers/<CONTAINER>`

Installed container image tag.

For instance, installation information of `etcd` is stored in
`<prefix>/containers/0/etcd` key.

## `<prefix>/install/<LRN>/debs/<DEBIAN_PACKAGE>`

Installed debian package version.

For instance, installation information of `etcdpasswd` is stored in
`<prefix>/debs/0/etcdpasswd` key.

## `<prefix>/status/current`

A leader of `neco-updater` creates and updates this key.

The value is a JSON object with these fields:

Name         | Type   | Description
----         | ----   | -----------
`version`    | string | Target `neco` version to be updated for all `servers`.
`servers`    | []int  | LRNs of current available boot servers under update. This is created using `<prefix>/bootservers`.
`stop`       | bool   | If `true`, `neco-worker` stops the update process.
`started_at` | string | Updating start time.

```json
{
    "version": "1.2.3-1",
    "servers": [1, 2, 3],
    "stop": false,
    "started_at": "2018-11-02T08:23:49.907839312Z"
}
```

`neco-worker` watches this key to start a new update process.
If `stop` becomes true, `neco-worker` should stop the ongoing update process immediately.

## `<prefix>/status/bootservers/<LRN>`

`neco-worker` creates and updates this key.

The value is a JSON object with these fields:

Name       | Type   | Description
----       | ----   | -----------
`version`  | string | Target `neco` version to be updated.
`step`     | int    | Current update step.
`finished` | bool   | Update to `version` successfully completed.
`error`    | bool   | If `true`, the update process was aborted due to an error.
`message`  | string | Description of an error.

```json
{
    "version": "1.2.3-1",
    "step": 2,
    "finished": false,
    "error": true,
    "message": "cke update failed"
}
```

`neco-updater` watches these keys to wait all workers to complete update process,
or detect errors during updates.

## `<prefix>/config/notification/slack`

The notification config to slack URL such as `https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXX`.

## `<prefix>/config/proxy`

HTTP proxy url to access Internet such as `https://squid.slack.com:3128`

## `<prefix>/config/check-update-interval`

Polling interval for checking new neco release in nanoseconds.

## `<prefix>/config/worker-timeout-duration`

Timeout from workers in nanoseconds.

## `<prefix>/vault-unseal-key`

Vault unseal key for unsealing automatically.

## `<prefix>/vault-root-token`

Vault root token for automatic setup for [dctest](../dctest/).
This key does not exist by default.
