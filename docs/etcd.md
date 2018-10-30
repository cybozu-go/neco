etcd schema
===========

## `<prefix>/leader/`

This prefix is used to elect a leader `neco-updater` who is responsible to invoke
update process to `neco-worker`.

## `<prefix>/bootservers`

This prefix is list of current available boot servers. This list is copied before 
starting the update process.

```json
["boot-1", "boot-2"]
```

## `<prefix>/current`

This prefix is information of latest `neco` version and current `<prefix>/bootservers`.

```json
{
    "version": "1.2.3-1",
    "servers": ["boot-1", "boot-2"]
}
```

Name      | Type     | Description
----      | ----     | -----------
`version` | string   | Target `neco` version to be updated for all `servers`.
`servers` | []string | Current available boot servers under update. This is created using `<prefix>/bootservers`.

## `<prefix>/notification**

**TODO**
This prefix is configuration of the notification service.

```json
{
    "webhook": "https://<webhook url>"
}
```

Name      | Type   | Description
----      | ----   | -----------
`webhook` | string | Slack web hook URL.

**TODO**
## `<prefix>/update/<version>`

This key is used to store the status of update process of `neco update-all`.

`<version>` is the debian package version of `neco`.

The value is a JSON object with these fields:

Name    | Type   | Description
------  | ------ | -----------
`abort` | bool   | If `true`, the update has been interrupted with an error.
`error` | string | Error message, when aborted.

## `<prefix>/update-leader/`

This prefix is used to elect a leader `neco update-all` process to update
programs gracefully.
