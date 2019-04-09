neco
====

`neco` is an interactive tool for administrators.

Features include:
* Bootstrap etcd and vault clusters.
* Edit etcd database to configure `neco-updater` and `neco-worker`.
* Initialize application data before installation.

Synopsis
--------

### Configure `neco-worker` and `neco-updater`

* `neco config set KEY VALUE`

    Change the setting for `KEY` to `VALUE`.
    Key and values are described in [another section](#config).

    Some special keys read their values from environment variables due to security concerns.
    In these cases, do not give `VALUE` in command line.

* `neco config get KEY`

    Show the current configuration for `KEY`.

### Boot server setup

* `neco setup [--no-revoke] [--proxy=PROXY] LRN [LRN ...]`

    Install and setup etcd cluster as well as Vault using given boot servers.
    `LRN` is the logical rack number of the boot server.  At least 3 LRNs
    should be specified.

    This command need to be invoked at once on all boot servers specified by LRN.

    When `--no-revoke` option is specified, it does not revoke the initial
    root token.  This is only for testing purpose.

    When `--proxy` option is specified, it uses this proxy to download container
    images. It also stores [`proxy` configuration](#configproxy) in the etcd database
    after it starts etcd, in order to run neco-updater and
    neco-worker with a proxy from the start.
    **DO NOT** pass `http_proxy` and `https_proxy` environment variables to `neco`.

* `neco init NAME`

    Initialize data for a new application.
    For example, this creates etcd user/role or Vault CA for the new application.

    This command should be **executed only once in the cluster**.

* `neco init-local NAME`

    Prepare files to start `NAME`.  For example, this issues a client certificate
    for etcd authentication.    This will ask users to input Vault username and
    password to issue certificates.

    This command should be **executed on all boot servers**.

* `neco init-data`

    Initialize data for sabakan and CKE. If uploaded versions are up to date, do nothing.
    This command must be invoked only once in the cluster after `neco init` and 
    `neco init-local` completed.

* `neco status`

    Show the status of the current update process.

* `neco join LRN [LRN ...]`

    Prepare certificates and files to add this server to the cluster.  
    `LRN` are a list of LRNs of the existing boot servers.

    To issue certificates, this command asks the user Vault username and password.
    If `VAULT_TOKEN` environment variable is not empty, it is used instead.

    This command also creates `/etc/neco/config.yml`.

    Etcd and Vault themselves are *not* installed by this command.  They are
    installed later by `neco-worker`.  Similarly, this command does not
    add the new server to etcd cluster.  `neco-worker` will add the server
    to etcd cluster.

* `neco leave LRN`

    Unregister `LRN` of the boot server from etcd.

* `neco recover`

    Removes the current update status from etcd to resolve the update failure.

### For worker nodes

* `neco ssh generate [--dump]`

    Generates a new SSH key pair for sabakan controlled machines.

    The generated public key is stored in etcd and will be automatically set for
    users defined in ignition templates.

    The generated private key is stored in Vault by using `ckecli vault ssh-privkey`.

    When `--dump` option is specified, the generated private key is also dumped
    to stdout.

### Vault related functions

* `neco vault unseal`

    Unseal the local vault server using the initial unseal key stored in etcd.

* `neco vault show-unseal-key`

    Show the initial vault unseal key if not removed.

* `neco vault remove-unseal-key`

    Remove the initial vault unseal key from etcd.

* `neco vault show-root-token`

    Show the initial root token, if not revoked during `neco setup`.

### BMC management functions

* `neco bmc config set KEY VALUE`

    Change the setting for `KEY` to `VALUE`.
    Keys and values are described below.

    - `bmc-user`: Register [`bmc-user.json`](https://github.com/cybozu-go/setup-hw/blob/master/README.md#etcnecobmc-userjson)
    - `ipmi-user`: Register IPMI username for power management.
    - `ipmi-password`: Register IPMI password for power management.

* `neco bmc config get KEY`

    Get the `VALUE` for `KEY`.

* `neco bmc setup-hw`

    Invoke `setup-hw` command in setup-hw container. If needed, reboot the machine.

* `neco ipmipower [start|stop|restart|status] SERIAL_OR_IP`

    Control power of a machine having `SERIAL` or `IP` address.

* `neco reboot-worker`

    Reboot all worker nodes.

### Miscellaneous

* `neco image NAME`

    Show docker image URL of `NAME` (e.g. "etcd", "coil", "squid").

* `neco completion`

    Dump bash completion rules for `neco` command.

<a name="config"></a>
Configurations
--------------

These configurations are stored in etcd database.

### `env`

Specify the cluster environment.
Possible values are: `staging` and `prod`.

`staging` environment will be updated with pre-releases of `neco` package.

Update never happens until this config is set.

### `slack`

Specify [Slack WebHook](https://api.slack.com/incoming-webhooks) URL.
`neco-updater` will post notifications to this.

<a name="configproxy"></a>
### `proxy`

Specify HTTP proxy server to access Internet.
It will be used by `neco-updater` and `neco-worker`.

### `quay-username`

Set username to authenticate to quay.io from `QUAY_USER` envvar.
It will be used by `neco-worker`.

### `quay-password`

Set password to authenticate to quay.io from `QUAY_PASSWORD` envvar.
It will be used by `neco-worker`.

### `check-update-interval`

Specify polling interval for checking new neco package release.
The value will be parsed by [`time.ParseDuration`][ParseDuration].

The default value is `10m`.

### `worker-timeout`

Specify timeout value to wait for workers during update process.
The value will be parsed by [`time.ParseDuration`][ParseDuration].

The default value is `60m`.

### `github-token`

Set GitHub personal access token for using GitHub API with authenticated user.
It will be used by `neco-updater` and `neco-worker`.

Use case
--------

### Setup three boot servers as initial cluster

1. Run `neco setup 0 1 2` on each boot server.
    1. Install etcd and vault.
    2. Start `vault` service temporarily to prepare CA and initial certificates
    3. Start TLS-enabled cluster.
    4. Restart `vault` as a real service, import CA to the `vault`.
    5. Reissue certificates for etcd and vault.
    6. Restart etcd and vault with new certificates.
    7. Save root token to the etcd key `<prefix>/vault-root-token`.
    8. Save new client certificates as `/etc/neco/etcd.crt` and `/etc/neco/etcd.key`
    9. Create `/etc/neco/neco-updater.yml` and `/etc/nec/neco-worker.yml`.
    10. Create an etcd key `<prefix>/vault-unseal-key`.
    11. Remove an etcd key `<prefix>/vault-root-token` by default.
    12. Add etcd key `<prefix>/bootservers/LRN` on the finished boot server.
2. Run `neco init NAME` on one of boot servers. etcd user/role has created.
3. Run `neco init-local NAME` on each boot server. Client certificates for `NAME` have issued.
4. Run `neco init-data` on one of boot servers.

### Add a new boot server

1. Run `neco join 0 1 2` on a new server.
    1. Install etcd and vault.
    1. Access another vault server to issue client certificates for etcd and vault.
    1. Save client certificates as `/etc/neco/etcd.crt` and `/etc/neco/etcd.key`
    1. Create `/etc/neco/neco-updater.yml` and `/etc/neco/neco-worker.yml`.
    1. Add member to the etcd cluster.
    1. Add a new boot server to the etcd key `<prefix>/bootservers/LRN`.
1. Run `neco init-local NAME` on a new boot server. Client certificates for `NAME` have issued.

Existing boot servers need to maintain application configuration files
to update the list of etcd endpoints.

### Setup a new program

When a new program is added to `artifacts.go`, it should be setup as follows:

0. `neco-worker` installs the program but does not start it yet.
1. Run `neco init NAME` on a boot server.
2. Run `neco init-local NAME` on all boot servers.

### Remove a dead boot server

1. Run `neco leave LRN` on the current running boot server.
    1. Remove etcd key `<prefix>/bootservers/LRN`.

Existing boot servers need to maintain application configuration files
to update the list of etcd endpoints.

[ParseDuration]: https://golang.org/pkg/time/#ParseDuration
