How automatic update works
==========================

`neco-update` updates packages in boot servers by following steps:

1. Check updates from GitHub releases of the `github.com/cybozu-go/neco`.
   Use latest package for each environement:
    - latest release including pre-releases if `/etc/neco/cluster` contains `stage*`
    - latest release excluding pre-releases if `/etc/neco/cluster` contains `prod*`
2. Update neco packages itself if the newer neco packages is released
    - Download debian package from GitHub Release and install it.
    - Restart `neco-updater` service.
3. Update sabakan contents
    - Elect a leader by `<prefix>/leader/` key in etcd.
    - Invoke `neco update-saba` to update sabakan contents.
4. Invoke `update-all`


`neco update-all`
-----------------

`neco update-all` sub-command updates applications with own strategy.
Strategies are described for each applications as following:

### CKE

1. Wait until CKE services stop in all boot servers
  - Stop CKE service in own boot server and commit state to etcd.
  - Watch all boot server commits stop state to etcd.
2. Update container image and restart `cke` service in all boot servers.

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

1. 大丈夫？

<!-- TODO Serf分断しないか？ -->

### etcdpasswd

Updater updates package and restart `ep-agent` service in any timing.
