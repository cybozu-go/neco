How automatic update works
==========================

Basic strategy
--------------

1. `neco-updater` elects a leader.
2. The leader checks new releases of `neco` package at GitHub periodically.
3. If a new release exists, the leader begins the update process.
4. Once the update completes, go to 2.
5. When the update fails, `neco-updater` sends alerts to admins and halts.

Problems and solutions
----------------------

What difficult in automatic updates of programs are:

* how to guarantee *all* boot servers reach a checkpoint.
* how to handle dead boot servers during update process.

For example, CKE must not be downgraded as it embeds Kubernetes versions.
If it was downgraded, Kubernetes cluster would be downgraded too.
To avoid this, all CKE processes must be stopped before starting update.

A possible solution is to register boot servers in etcd.
This way, the update process can recognize existing boot servers.

To settle the second, dead servers need to be unregistered from etcd.

Further corner cases are discussed below.

### Adding a boot server during update process

The update process keeps a copy of the current list of boot servers
at the beginning.  During update process, new boot servers are ignored.

### A boot server dies during update process

The update process should be aborted due to some timeouts.  To recover,
the administrator need to unregister the dead server from etcd, then
restart `neco-updater`.

`neco-updater` retries the update process with the new list of boot servers.

Tag name and release flow
-------------------------

`neco-updater` watches newer version in GitHub releases by comparing its tags.
The tag MUST contains version number with prefix.
The prefix is used to filter build target on CI/CD rules, so `neco-updater`
does not consider prefix and compares version number after first hyphen.

For example, a tag name `release-2018.11.07-1` consist of prefix `release-` and
version number `2018.11.07-1`.  `neco-updater` ignores prefix `release-` and
use the version number `2018.11.07-1` on a comparison of the tags.

`neco-updater` switches downloading version by data-center environment.  User
must select the environment by `neco config set env ENV` sub-command.
If `production` is set, `neco-updater` downloads latest release excluding
pre-release version.  Otherwise if `staging` is set, `neco-updater` downloads
latest version from among releases and pre-releases.

Implementation of update process
--------------------------------

`neco-updater` services is responsible to check if a new neco version exists,
control workers, and notify the results of the update.  `neco-updater` does
leader election and only one process works.

`neco-worker` service responsible to do installing and updating application on
each nodes.  The detailed update process follows below steps:

1. Check latest version from GitHub releases.
2. The leader `neco-updater` takes a copy of the list of boot servers.
3. The leader puts the new version of `neco` package in etcd.  `<prefix>/status/current`

    ```json
    {
        "version": "1.2.3-1",
        "servers": [1, 2, 3],
        "stop": false,
    }
    ```

4. On each boot server, `neco-worker` watches etcd; when it finds 2, install the new version.
5. If `neco-worker` is the same version, it starts update process.
6. Before each update step, `neco-worker` synchronizes with other workers.

    ```json
    {
        "version": "1.2.3-1",
        "step": 2,
        "finished": false,
        "error": false,
        "message": ""
    }
    ```

To synchronize with other workers, `neco-worker` records the current step in
`step` field of `<prefix>/status/bootserver/<LRN>` status record.  Once all
workers record the new step, they proceed to the step.

If it takes too long, `neco-worker` should time-outs.

Failure and recovery
--------------------

When something goes wrong, the update process need to be aborted.

`neco-worker` inform `neco-updater` of an error during update process
through etcd key `<prefix>/status/bootserver/<LRN>`.

`neco-updater` share any errors with `neco-worker` by updating `stop`
field of etcd key `<prefix>/status/current` `stop` to `true`.

To recover from failures, `neco recover` removes these keys from etcd.
Then `neco-updater` re-creates `<prefix>/status/current` etcd key to restart jobs.

`neco-worker`
-------------

`neco-worker` installs/updates applications as follows:

### CKE

1. Wait until CKE services stop in all boot servers.
  - Stop CKE service in own boot server and commit state to etcd.
  - Watch all boot server commits stop state to etcd.
2. Update container image and restart `cke` service in all boot servers.

**Note**:
CKE must always be stopped regardless of whether it is the latest version or not.
Otherwise, other `neco-worker` may wait forever at this check point.

### Vault

1. Wait until Vault services stop in all boot servers.
  - Stop Vault service in own boot server and commit state to etcd.
  - Watch all boot server commits stop state to etcd.
2. Update container image and restart `vault` service in all boot servers.
3. Unseal vault.

### etcd

Updater elects one leader and does rolling update etcd members.

1. Elect leader updater.
2. Checks if own etcd can be updated.
3. Update etcd archive and restart a service.
4. Resign leader updater.

### sabakan

Updater updates container image and restart `sabakan` in any timing.

### sabakan contents e.g. container images, OS images and ignitions

1. Elect leader updater. Only a leader does procedure as follows.
2. Check `<prefix>/worker/sabakan-content`.

    1. If `version` matches with current neco version and `success` is true, worker does nothing.
    2. If `version` matches and `success` is false, worker aborts the process.
    3. If `version` does not match or key does not exist, worker does procedure as follows.

3. Check if each sabakan content can be updated. If so, download artifacts, then upload them to sabakan.
4. Put `<prefix>/status/sabakan-content` with value.

If on failure, `neco recover` command removes also this key.

### setup-hw

Updater updates container image and restart setup-hw service in any timing.

### Serf

1. Elect leader updater.
2. Checks if own serf can be updated.
3. Update serf archive and restart a service.
4. Resign leader updater.

### etcdpasswd

Updater updates package and restart `ep-agent` service in any timing.
