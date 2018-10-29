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

Implementation of update process
--------------------------------

1. The leader `neco-updater` takes a copy of the list of boot servers.
2. The leader puts the new version of `neco` package in etcd.  `<prefix>/current`

    ```json
    {
        "version": "1.2.3-1",
        "servers": ["boot-1", "boot-2"]
    }
    ```

3. On each boot server, `neco-worker` watches etcd; when it finds 2, install the new version.
4. If `neco-worker` is the same version, it starts update process.

When `neco-worker` needs to synchronize with others, it uses an etcd key as a counter.
Once the counter becomes the same number of boot servers, `neco-worker` proceeds.
If it takes too long, `neco-worker` should time-outs.

`neco update-all`
-----------------

`neco update-all` sub-command updates applications with own strategy.
Strategies are described for each applications as following:

### CKE

1. Wait until CKE services stop in all boot servers
  - Stop CKE service in own boot server and commit state to etcd.
  - Watch all boot server commits stop state to etcd.
2. Update container image and restart `cke` service in all boot servers.

**Note**:
CKE must always be stopped regardless of whether it is the latest version or not.
Otherwise, other `neco-worker` may wait forever at this check point.

### Vault

1. Wait until Vault services stop in all boot servers
  - Stop Vault service in own boot server and commit state to etcd.
  - Watch all boot server commits stop state to etcd.
2. Update container image and restart `vault` service in all boot servers.
3. Unseal vault.

### etcd

Updater elects one leader and does rolling update etcd members.

1. Elect leader updater
2. Checks if own etcd can be updated
3. Update etcd archive and restart a service
4. Resign leader updater

### sabakan

Updater updates container image and restart `sabakan` in any timing.

### OMSA

Updater updates container image and restart OMSA service in any timing.

### Serf

1. Elect leader updater
2. Checks if own serf can be updated
3. Update etcd archive and restart a service
4. Resign leader updater

### etcdpasswd

Updater updates package and restart `ep-agent` service in any timing.
