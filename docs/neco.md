neco
====

`neco` is an interactive tool for administrators.

Features include:
* Bootstrap etcd and vault clusters.
* Edit etcd database to configure `neco-updater` and `neco-worker`.
* Initialize application data before installation.

- [Synopsis](#synopsis)
  - [Configure `neco-worker` and `neco-updater`](#configure-neco-worker-and-neco-updater)
  - [Boot server setup](#boot-server-setup)
  - [For worker nodes](#for-worker-nodes)
  - [Vault related functions](#vault-related-functions)
  - [BMC management functions](#bmc-management-functions)
  - [CKE related functions](#cke-related-functions)
  - [TPM related functions](#tpm-related-functions)
  - [Automated firmware application functions](#automated-firmware-application-functions)
  - [Session log recording](#session-log-recording)
  - [Miscellaneous](#miscellaneous)
- [Configurations](#configurations)
  - [`env`](#env)
  - [`slack`](#slack)
  - [`proxy`](#proxy)
  - [`check-update-interval`](#check-update-interval)
  - [`worker-timeout`](#worker-timeout)
  - [`github-token`](#github-token)
  - [`node-proxy`](#node-proxy)
  - [`external-ip-address-block`](#external-ip-address-block)
- [Use case](#use-case)
  - [Setup three boot servers as initial cluster](#setup-three-boot-servers-as-initial-cluster)
  - [Add a new boot server](#add-a-new-boot-server)
  - [Setup a new program](#setup-a-new-program)
  - [Remove a boot server](#remove-a-boot-server)

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

* `neco is-running IMAGE`

    Check if the given `IMAGE` is running as a container on the boot server.
    If it is running, this exits with status 0.  Otherwise, with status 1.

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
    - `repair-user`: Register BMC username for repair operations.
    - `repair-password`: Register BMC password for repair operations.

* `neco bmc config get KEY`

    Get the `VALUE` for `KEY`.

* `neco bmc repair BMC_TYPE BMC_specific_command...`

    Try to repair an unhealthy/unreachable machine by invoking BMC functions remotely.

    * Dell iDRAC:
        * `neco bmc repair dell reset-idrac SERIAL_OR_IP`
            Reset the iDRAC of a machine having `SERIAL` or `IP` address.
        * `neco bmc repair dell discharge SERIAL_OR_IP`
            Simulate power-disconnection and discharge of a machine having `SERIAL` or `IP` address. This implies reboot of the machine.

* `neco bmc setup-hw`

    Invoke `setup-hw` command in setup-hw container. If needed, reboot the machine.

* `neco power [start|stop|restart|status] [--wait-for-stop] SERIAL_OR_IP`

    Control power of a machine having `SERIAL` or `IP` address. It just request BMC to control power, not wait for its completion.

    When `--wait-for-stop` option is specified for `stop` or `restart` action, it wait until the machine stops.

* `neco reboot-and-wait SERIAL_OR_IP`

    Reboot a machine having `SERIAL` or `IP` address, and wait for its boot-up.

* `neco reboot-check SERIAL_OR_IP UNIXTIME`

    Check (re)boot-up of a machine having `SERIAL` or `IP` address after the `UNIXTIME`.
    If rebooted, prints `true`. If not rebooted, prints `false`.

* `neco reboot-worker`

    Reboot all or specified worker nodes.

    This uses CKE's function of [graceful reboot](https://github.com/cybozu-go/cke/blob/main/docs/reboot.md) for the nodes used by CKE.
    As for the other nodes, this reboots them immediately.
    If some nodes are already powered off, this command does not do anything to those nodes.
    [`sabactl machines get`-like options](https://github.com/cybozu-go/sabakan/blob/main/docs/sabactl.md#sabactl-machines-get-query_param) can be used to narrow down the machines to be rebooted.

### CKE related functions

The name of the cluster in [cke-template.yml](../etc/cke-template.yml) will be overwritten with the value read from `/etc/neco/cluster`.

The weight is values of each role for **overriding** `labels["cke.cybozu.com/weight"]` values in [cke-template.yml](../etc/cke-template.yml).
When commands as follows run `ckecli sabakan set-template` internally, read etcd saved weight values and then generate `cke-template.yml`.

- `neco-worker`
- `neco init-data`
- `neco cke update`

See details [Role and weights](https://github.com/cybozu-go/cke/blob/main/docs/sabakan-integration.md#roles-and-weights).

* `neco cke weight list`

    List current weight of roles.

* `neco cke weight get ROLE`

    Get current weight of given role.

* `neco cke weight set ROLE WEIGHT`

    Set given weight to the role.

* `neco cke update`

    Update cke template using overriding weights. This is useful if administrator updates role and weights in the running Kubernetes cluster.

### TPM related functions

* `neco tpm clear SERIAL_OR_IP`

Clear TPM devices on a machine having `SERIAL` or `IP` address.
The command fails when the target machine's status is not retiring.
`--force` option is explicitly required.

* `neco tpm show SERIAL_OR_IP`

Show TPM devices on a machine having `SERIAL` or `IP` address.

### Automated firmware application functions

* `neco isoreboot ISO_FILE`

Connect iso file to Virtual DVD and schedule reboot.

[`sabactl machines get`-like options](https://github.com/cybozu-go/sabakan/blob/main/docs/sabactl.md#sabactl-machines-get-query_param) can be used to narrow down the machines to be updated.

* `neco apply-firmware UPDATER_FILE...`

Send firmware updaters to BMC and schedule reboot.

[`sabactl machines get`-like options](https://github.com/cybozu-go/sabakan/blob/main/docs/sabactl.md#sabactl-machines-get-query_param) can be used to narrow down the machines to be updated.

### Session log recording

* `neco session-log start`

Start session logging by script(1). After invoked shell exits, session log is put to the object bucket located at http://s3gw.session-log.svc .

### Miscellaneous

* `neco image NAME`

    Show docker image URL of `NAME` (e.g. "etcd", "coil", "squid").

* `neco teleport config`

    Generate config for teleport by filling template with secret in file and dynamic info in etcd.

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

### `node-proxy`

Specify HTTP proxy server to access Internet for worker nodes.
This value is used as metadata in the ignition template.

### `external-ip-address-block`

Specify an IP address block assigned to Nodes by a LoadBalancer controller.
This value is used as metadata in the ignition template.

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
4. (Optional) Run `neco cke weight` on one of boot servers for generating `cke-template.yml`.
5. Run `neco init-data` on one of boot servers.

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

### Remove a boot server

1. Run `neco leave LRN` on the current running boot server.
    1. Remove etcd key `<prefix>/bootservers/LRN`.

Existing boot servers need to maintain application configuration files
to update the list of etcd endpoints.

[ParseDuration]: https://golang.org/pkg/time/#ParseDuration
