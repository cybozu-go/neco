Vault schema
============

## `ca/server`

`pki` is mounted.
Used to issue server certificates.

## `ca/boot-etcd-peer`

`pki` is mounted.
Used to issue etcd peer certificates.

## `ca/boot-etcd-client`

`pki` is mounted.
Used to issue etcd client certificates for Vault.

## `secret`

`kv` version 1 is mounted.
Used for general key-value secrets.

### `secret/bootstrap`

Used to send signal from the leader to the others in bootstrap.

### `secret/bootstrap_done/<LRN>`

Used to send signal from non-leaders to the leader in bootstrap.

### `secret/teleport`

Used for a teleport token sent from node/proxy to auth.
