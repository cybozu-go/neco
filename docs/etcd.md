etcd schema
===========

Legend:
* `<prefix>` = `/neco/`

## `<prefix>/leader/updater/`

This prefix is used to elect a leader `neco-updater` who is responsible to invoke
update process to `neco-worker`.

## `<prefix>/leader/worker/`

This prefix is used to elect a leader `neco-worker`.

## `<prefix>/info/bootservers/<LRN>`

This prefix is current available boot servers. Current available boot server is
registered as a key `LRN`.  The value is empty.

## `<prefix>/info/neco-release`

This key stores the Debian package version of `neco` released at GitHub.

## `<prefix>/info/ssh-pubkey`

The value is a SSH public key.  This key will be used to authenticate
users in nodes.

## `<prefix>/contents/<target>`

`<target>` includes `sabakan`, `cke`, etc.

The value is a JSON object with these fields:

| Name      | Type   | Description                                                    |
| --------- | ------ | -------------------------------------------------------------- |
| `version` | string | sabakan contents version (the same as `neco` package version). |
| `success` | bool   | If `true`, update succeeded.                                   |

```json
{
    "version": "1.2.3-1",
    "success": false
}
```

`neco-worker` checks this key to decide procedure for updating contents for `<target>`.
See details [update.md](update.md#sabakan-contents-eg-container-images-os-images-and-ignitions).

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

| Name         | Type   | Description                                                                                        |
| ------------ | ------ | -------------------------------------------------------------------------------------------------- |
| `version`    | string | Target `neco` version to be updated for all `servers`.                                             |
| `servers`    | []int  | LRNs of current available boot servers under update. This is created using `<prefix>/bootservers`. |
| `stop`       | bool   | If `true`, `neco-worker` stops the update process.                                                 |
| `started_at` | string | Updating start time.                                                                               |

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

| Name      | Type   | Description                                                                      |
| --------- | ------ | -------------------------------------------------------------------------------- |
| `version` | string | Target `neco` version to be updated.                                             |
| `step`    | int    | Current update step.                                                             |
| `cond`    | int    | [`UpdateCondition`](https://godoc.org/github.com/cybozu-go/neco#UpdateCondition) |
| `message` | string | Description of an error.                                                         |

```json
{
    "version": "1.2.3-1",
    "step": 2,
    "cond": 0,
    "message": "cke update failed"
}
```

`neco-updater` watches these keys to wait all workers to complete update process,
or detect errors during updates.

## `<prefix>/config/notification/slack`

The notification config to slack URL such as `https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXX`.

## `<prefix>/config/proxy`

HTTP proxy url to access Internet such as `https://squid.slack.com:3128`

## `<prefix>/config/ghcr-username`

User name to authenticate ghcr.io.

## `<prefix>/config/ghcr-password`

Password to authenticate ghcr.io.

## `<prefix>/config/check-update-interval`

Polling interval for checking new neco release in nanoseconds.

## `<prefix>/config/worker-timeout-duration`

Timeout from workers in nanoseconds.

## `<prefix>/config/github-token`

GitHub personal access token.

## `<prefix>/config/node-proxy`

HTTP proxy url for worker nodes to access Internet such as `http://squid:3128`.

## `<prefix>/config/external-ip-address-block`

IP address block to to be published externally.

## `<prefix>/vault-unseal-key`

Vault unseal key for unsealing automatically.

## `<prefix>/vault-root-token`

Vault root token for automatic setup for [dctest](../dctest/).
This key does not exist by default.

## `<prefix>/bmc/bmc-user`

`bmc-user.json` contents.

## `<prefix>/bmc/ipmi-user`

IPMI username for power management.

## `<prefix>/bmc/ipmi-password`

IPMI password for power management.

## `<prefix>/teleport/auth-token`

Token for accessing to teleport auth server

## `<prefix>/cke/weight`

A JSON object with these fields:

| Name     | Type  | Description       |
| -------- | ----- | ----------------- |
| `<ROLE>` | float | A role of weight. |

This is used when `neco-worker` and `neco init-data` run `ckecli sabakan set-template`.

```json
{
    "cs": 18,
    "ss": 10
}
```
