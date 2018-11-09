neco
====

`neco` is an interactive tool for administrators.

Features include:
* Bootstrap etcd and vault clusters.
* Edit etcd database to configure `neco-updater` and `neco-worker`.
* Initialize application data before installation.

Synopsis
--------

* `neco config set KEY VALUE`

    Change the setting for `KEY` to `VALUE`.
    Key and values are described in [the next section](#config).

* `neco config get KEY`

    Show the current configuration for `KEY`.

* `neco init NAME`

    Initialize data for a new application.
    For example, this creates etcd user/role or Vault CA for the new application.

    This command should be **executed only once in the cluster**.

* `neco init-local [--start] NAME`

    Prepare files to start `NAME`.  For example, this issues a client certificate
    for etcd authentication.    This will ask users to input Vault username and
    password to issue certificates.

    This command should be **executed on all boot servers**.

    If `--start` is given, the program is started after initialization.

* `neco join`

    Join this server as a new boot server and an etcd member.
    It asks vault user and password to generate a vault token, then issue client
    certificates for etcd and vault for a new boot server.

* `neco leave LRN`

    Unregister `LRN` of the boot server from etcd.

* `neco recover`

    Removes the current update status from etcd to resolve the update failure.

* `neco setup [--no-revoke] LRN [LRN ...]`

    Install and setup etcd cluster as well as Vault using given boot servers.
    `LRN` is the logical rack number of the boot server.  At least 3 LRNs
    should be specified.

    This command need to be invoked at once on all boot servers specified by LRN.

    When `--no-revoke` option is specified, it does not revoke the initial
    root token.  This is only for testing purpose.

* `neco status`

    Show the status of the current update process.

* `neco vault unseal`

    Unseal the local vault server using the initial unseal key stored in etcd.

* `neco vault show-unseal-key`

    Show the initial vault unseal key if not removed.

* `neco vault remove-unseal-key`

    Remove the initial vault unseal key from etcd.

* `neco vault show-root-token`

    Show the initial root token, if not revoked during `neco setup`.

<a name="config"></a>
Configurations
--------------

These configurations are stored in etcd database.

### `env`

Specify the cluster environment.  The default is `staging`.
Possible values are: `staging` and `prod`.

`staging` environment will be updated with pre-releases of `neco` package.

### `slack`

Specify [Slack WebHook](https://api.slack.com/incoming-webhooks) URL.
`neco-updater` will post notifications to this.

### `proxy`

Specify HTTP proxy server to access Internet.
It will be used by `neco-updater` and `neco-worker`.

### `check-update-interval`

Specify polling interval for checking new neco package release.
The value will be parsed by [`time.ParseDuration`][ParseDuration].

The default value is `10m`.

### `worker-timeout`

Specify timeout value to wait for workers during update process.
The value will be parsed by [`time.ParseDuration`][ParseDuration].

The default value is `60m`.

Use case
--------

### Setup three boot servers as initial cluster

1. Run `neco setup 0 1 2` on each boot server.
    1. Install etcd and vault.
    1. Start `vault` service temporarily to prepare CA and initial certificates
    1. Start TLS-enabled cluster.
    1. Restart `vault` as a real service, import CA to the `vault`.
    1. Reissue certificates for etcd and vault.
    1. Restart etcd and vault with new certificates.
    1. Save root token to the etcd key `<prefix>/vault-root-token`.
    1. Save new client certificates as `/etc/neco/etcd.crt` and `/etc/neco/etcd.key`
    1. Create `/etc/neco/neco-updater.yml` and `/etc/nec/neco-worker.yml`.
    1. Create an etcd key `<prefix>/vault-unseal-key`.
    1. Remove an etcd key `<prefix>/vault-root-token` by default.
    1. Add etcd key `<prefix>/bootservers/LRN` on the finished boot server.
1. Run `neco init NAME` on one of boot servers. etcd user/role has created.
1. Run `neco init-local NAME` on each boot server. Client certificates for `NAME` have issued.
1. Run `systemctl start neco-worker.service` to install applications.
     1. Create configuration file and systemd unit files for each applications.
     1. Download container images and deb packages.
     1. Install applications, and start them.

### Add a new boot server

1. Run `neco join 0 1 2` on a new server.
    1. Install etcd and vault.
    1. Access another vault server to issue client certificates for etcd and vault.
    1. Save client certificates as `/etc/neco/etcd.crt` and `/etc/neco/etcd.key`
    1. Create `/etc/neco/neco-updater.yml` and `/etc/neco/neco-worker.yml`.
    1. Add member to the etcd cluster.
    1. Add a new boot server to the etcd key `<prefix>/bootservers/LRN`.
1. Run `neco init-local NAME` on a new boot server. Client certificates for `NAME` have issued.
1. Run `systemctl start neco-worker.service` to install applications.
    1. Create etcd configuration file with a new member.
    1. Start etcd.
    1. Create vault configuration file with a new member.
    1. Start vault.
    1. Unseal vault.
    1. Install and start other applications.

Existing boot servers need to maintain application configuration files
to update the list of etcd endpoints.

### Setup a new program

When a new program is added to `artifacts.go`, it should be setup as follows:

0. `neco-worker` installs the program but does not start it yet.
1. Run `neco init NAME` on a boot server.
2. Run `neco init-local --start NAME` on all boot servers.

### Remove a dead boot server

1. Run `neco leave LRN` on the current running boot server.
    1. Remove etcd key `<prefix>/bootservers/LRN`.

Existing boot servers need to maintain application configuration files
to update the list of etcd endpoints.

[ParseDuration]: https://golang.org/pkg/time/#ParseDuration
