etcd schema
===========

## `<prefix>/leader/`

This prefix is used to elect a leader `neco-updater` who is responsible to invoke
update process to `neco-worker`.

## `<prefix>/bootservers/LRN`

This prefix is current available boot servers. Current available boot server is
registered as a key `LRN`.

## `<prefix>/current`

This prefix is information of latest `neco` version and current `<prefix>/bootservers`.

A leader of `neco-updater` creates this key.

```json
{
    "version": "1.2.3-1",
    "servers": [1, 2, 3]
}
```

Name      | Type   | Description
----      | ----   | -----------
`version` | string | Target `neco` version to be updated for all `servers`.
`servers` | []int  | LRNs of current available boot servers under update. This is created using `<prefix>/bootservers`.

## `<prefix>/notification`

This prefix is configuration of the notification service.

```json
{
    "webhook": "https://<webhook url>"
}
```

Name      | Type   | Description
----      | ----   | -----------
`webhook` | string | Slack web hook URL.

## `<prefix>/vault-unseal-key`

Vault unseal key for unsealing automatically.

## `<prefix>/vault-root-token`

Vault root token for automatic setup for [dctest](../dctest/).
This key does not exist by default.

## `<prefix>/stop`

`neco-worker` stop update process if the key exists.
