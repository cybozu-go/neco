neco
====

`neco` is an interactive tool for administrators.

It installs/updates miscellaneous programs as well as maintaining etcd database.

Usage
-----

```console
# neco [OPTIONS]
```

Options
-------

Options are defined by [cybozu-go/etcdutil](https://github.com/cybozu-go/etcdutil), and not shown below will use default values of the etcdutil.

Synopsis
--------

* `neco config set env staging|prod`

    Set cluster environment to use pre-released or released neco packages from
    GitHub releases.  Latest pre-released is used if `staging` is set.
    Otherwise, latest released is used if `prod` is set.

* `neco config set slack URL`

    Set Slack webhook URL for notification from `neco-updater`.

* `neco config set proxy URL`

    Set proxy URL to access Internet.

* `neco config set check-update-duration DURATION`

    Set duration for checking new neco release. DURATION format must be parsable by
    [`time.ParseDuration`](https://golang.org/pkg/time/#ParseDuration) such as `60m`.

* `neco setup [--no-revoke] LRN [LRN ...]`

    Install and setup etcd cluster as well as Vault using given boot servers.
    `LRN` is the logical rack number of the boot server.  At least 3 LRNs
    should be specified.

    This command should be invoked at once on all boot servers specified by LRN.

    When `--no-revoke` option is specified, it does not remove the etcd key
    `<prefix>/vault-root-token`. This option is used by automatic setup of
    [dctest](../dctest).

* `neco init NAME`

    Initialize data for new application of the cluster.  
    Setup etcd user/role for a new application `NAME`. This command should not
    be executed more than once.

* `neco init-local [--start] NAME`

    Initialize data for new application of a boot server executes. This command
    should not be executed more than once.  
    It asks vault user and password to generate a vault token, then issue client
    certificates for new a application `NAME`.

    If `--start` is given, the program is started after installation.

* `neco join LRN [LRN ...]`

    Join this server as a new boot server and an etcd member.
    `LRN` is the logical rack number of current available boot servers.
    It asks vault user and password to generate a vault token, then issue client
    certificates for etcd and vault for a new boot server.

* `neco leave LRN`

    Unregister `LRN` of the boot server from etcd.

* `neco recover`

    Removes the current update status from etcd to resolve the update failure.

* `neco vault unseal`

    Unseal the local vault server using the initial unseal key stored in etcd.

* `neco vault show-unseal-key`

    Show the initial vault unseal key if not removed.

* `neco vault remove-unseal-key`

    Remove the initial vault unseal key from etcd.

* `neco vault show-root-token`

    Show the initial root token, if not revoked during `neco setup`.

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


`neco-updater` and `neco-worker` would no longer to update on the dead boot server.
