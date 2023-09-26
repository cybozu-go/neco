How Flatcar Container Linux boots in Neco
========================================

Ignition is a provisioning system for Flatcar Container Linux.
Users can do virtually anything during boot process by writing systemd units.

This document describes how Neco constructs Ignition configurations and what
the Ignition does during the boot process of Container Linux.

Resources
---------

* [Ignition template system](https://github.com/cybozu-go/sabakan/blob/master/docs/ignition_template.md)

    The specification document of the template system for Ignition configurations.
    It is a feature of [sabakan][].

* [`ignitions/common/common.yml`](../ignitions/common/common.yml)

    This is the ignition template.  The template references files in
    [`ignitions/common/`](../ignitions/common/) directory.
    To place different files for each environment, create a new `common-<cluster>.yml` and specify it in `site-<cluster>.yml`.
    For information on how to generate the file, please refer to the [ignition-template](./ignition-template.md) documentation.


* [Ignition documentation](https://www.flatcar.org/docs/latest/provisioning/ignition/)

    The current template is written in [spec version 2.2](https://coreos.com/ignition/docs/latest/configuration-v2_2.html).

[sabakan]: https://github.com/cybozu-go/sabakan

Boot process
------------

The boot process runs roughly as follows:

1. Configure network with DHCP
2. Setup dm-crypt volumes using `sabakan-cryptsetup`
3. Prepare LVM volumes on encrypted disks
4. Mount LVM volumes under `/var`
5. Reconfigure network with BIRD
6. Run `chronyd` to synchronize time
7. Start docker
8. Configure BIOS and BMC
9. Run `serf` and other programs as a docker container

There are clear and strong reasons why the process is ordered this way.

### Configure network with DHCP

* Why the network need to be configured with DHCP and later be re-configured?

    In Neco, each server has two network links connected to ToR switches.
    These links will finally have _link-local scope IP addresses_ to hide them
    from other servers.  Each server has a single global scope IP address and
    announces it using BGP.

    However, this final configuration cannot be configured easily within
    ignition because speaking BGP requires BIRD and it needs to be run as a
    container.  To run a container, the image for the container must be
    pulled first which requires some basic networking.

* What are requirements for DHCP configured network?

    The network need to be configured to communicate with sabakan server to
    download `sabakan-cryptsetup` utility.  Since the sabakan server may exist
    in another network, the (default) gateway must also be configured.

### Setup dm-crypt volumes using `sabakan-cryptsetup`

* What is `sabakan-cryptsetup`?

    `sabakan-cryptsetup` is a disk encryption utility using dm-crypt.
    It generates a disk encryption key, and encrypts the encryption key to
    send the key to sabakan.  At the next boot, it downloads the encrypted
    disk encryption key and decrypt the key to decrypt the disk.

* What devices are encrypted?

    Any persistent storage such as NVMe SSD, HDD, or [BOSS][].

### Prepare LVM volumes on encrypted disks

* Why LVM is used?

    Because the number and capacity of disks may vary between servers.
    Using LVM makes it easy to adjust volume sizes based on demands.

### Mount LVM volumes under `/var`

* How logical volumes will be mounted?

    Each logical volume is labelled, and the label is referenced from
    `/etc/fstab`.  This way, Linux kernel detects the label and
    mounts the volume following `/etc/fstab`.

* Why `/etc/fstab` is used?  Can `*.mount` systemd unit be used instead?

    `/etc/fstab` is used to specify `x-systemd.device-timeout=600` parameter.
    This parameter can only be specified in `/etc/fstab`, not in mount units.

* Why `x-systemd.device-timeout` need to be specified?

    To extend waiting duration for mounting block devices to become available.
    `sabakan-cryptsetup` takes several minutes; without this option, mount
    and following boot process would fail.

* What directories are mounted, and why?

    Currently, the following directories are mounted to keep data on disks.
    `/var/lib/systemd` is mounted to persist `pstore` that stores logs created
    on a kernel panic, whereas the rest are mounted because the size of data in
    these directories is large.

    - `/var/lib/k8s-containerd`
    - `/var/lib/docker`
    - `/var/lib/kubelet`
    - `/var/lib/systemd`

### Reconfigure network with BIRD

* How and why network is reconfigured?

    Network is reconfigured by overwriting `/etc/systemd/network/01-eth*.network`
    files then restart `systemd-networkd.service`.  DHCP configurations are
    cleared and the two network links get link-local IP addresses.

    Routing information is obtained by running BIRD as a BGP speaker.
    It also announces a global scoped IP address of the server via BGP.

    After reconfiguration, the server gets redundant links to its global
    scoped IP address thanks to ECMP.  This is the reason.

* How BIRD is run?

    BIRD is run as a container.

### Run `chronyd` to synchronize time

* Why `chronyd` need to be run before docker?

    This is because running docker will start tons of other services
    that should run after time synchronization.

### Start docker

* How docker is configured to run after time synchronization?

    By overriding unit ordering by adding drop-in files under
    `/etc/systemd/system/docker.{socket,service}.d`.

### Configure BIOS and BMC

* How BIOS and BMC are configured?

    By running [`setup-hw`](https://github.com/cybozu-go/setup-hw) tool in a docker container.

* Why BIOS and BMC need to be configured before running serf and other services?

    Because changing BIOS configurations sometimes require server reboot.

### Run `serf` and other programs as a docker container

* Why `serf` is run last?

    Because running `serf` will tell sabakan that the machine becomes running
    and available.


systemd targets and ordering dependencies
-----------------------------------------

To implement the boot process described in the previous section, we need to
know and carefully program ordering dependencies between systemd units.

First, learn these systemd special targets:

* `basic.target`: this target is reached after local filesystem is mounted.
* `network.target`: this target is reached after `local-fs.target` and `systemd-networkd.service`.
* `network-online.target`: this target is reached after the network is fully configured.
* `time-sync.target`: this target is reached after the time is synchronized with NTP servers.

Ordering dependency between `network.target` and `local-fs.target` is indirect.
`network.target` runs after `systemd-resolved.service` which runs after `systemd-tmpfiles-setup.service` which runs after `local-fs.target`.  Because of this dependency, _running after `network.target` means that the service runs after encrypted local volumes are mounted_.

Mounting encrypted local volumes under `/var` requires that services to encrypt disks and create LVM volumes need to be run before `basic.target`.  Because services have default ordering dependency that run after `basic.target`, these services must declare `DefaultDependencies=no`.

Specifically, following services have `DefaultDependencies=no`:

* `neco-wait-dhcp-online.service`: to wait IP address is configured by DHCP
* `sabakan-cryptsetup.service`: to prepare encrypted disks
* `setup-var.service`: to prepare LVM volumes

After mounting volumes, next thing to do is to reconfigure the network.
This is done by running following services before `network-online.target`:

* `bird-wait.service`: to wait for receiving routes via BGP.
* `disable-nic-offload.service`: to disable Intel NIC checksum offloading (for stability).

Others are run after `network-online.target`.

Masked services
---------------

Following services are masked:

* `update-engine.service`: we will use other system to update Container Linux.
* `locksmithd.service`: ditto.
* `update-engine-stub.timer`: ditto.
* `ntpd.service`: we use chrony.
* `systemd-timesyncd.service`: ditto.
* `rkt-metadata.service`: we do not use this.
* `rkt-metadata.socket`: ditto.
* `iscsiuio.service`: we do not use this.
* `iscsiuio.socket`: ditto.

Serf tags
---------

Container Linux will periodically update serf tags as follows:

| Name                   | Description                               |
| ---------------------- | ----------------------------------------- |
| `serial`               | The serial code of the machine.           |
| `os-name`              | "Flatcar Container Linux by Kinvolk"      |
| `os-version`           | Container Linux version.                  |
| `uptime`               | Output of `uptime` command.               |
| `systemd-units-failed` | Failed unit names.                        |
| `version`              | Version of `neco` that boot this machine. |

[BOSS]: https://www.dell.com/support/article/us/en/04/sln310144/boss-s1-card
