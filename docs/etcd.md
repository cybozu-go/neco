etcd schema
===========

## `<prefix>/leader/`

This prefix is used to elect a leader `neco-updater` who is responsible to invoke
`neco update-saba`.

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
