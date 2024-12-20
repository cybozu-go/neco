Boot server installer
=====================

`installer/` directory contains source code and `Makefile` to build a custom
Ubuntu installer for Neco boot servers.

The installer automates almost all installation processes except for storage.
It also installs [BIRD][] and [chrony][] that are necessary to configure
networking.

For post-installation setup, a program is installed as `/extras/setup`.

## How auto-installation works

The custom installer makes use of [the autoinstall feature of the official
live server image](https://ubuntu.com/server/docs/install/autoinstall-quickstart).

The auto-installation script is [`installer/autoinstall/user-data`](../installer/autoinstall/user-data).
The script leaves storage configuration as interactive because we need to manually
input disk encryption key.

After installation, the system allows users to sign-in using `cybozu`/`cybozu` as user/password.
Make sure to change or lock the password.

## How to build the custom installer

1. Prepare `installer/cluster.json` file.

    An example is available at [`installer/cluster.json.example`](../installer/cluster.json.example).

2. Run `make setup` in `installer/` directory.
3. Run `make iso` in `installer/` directory.

The custom installer will be built as `installer/build/cybozu-*.iso`.

## Post-installation setup

After installation, the boot server needs to be configured with `/extras/setup`.
It needs to take an HTTP proxy server address via `http_proxy` environment variable.
If the environment variable is not set, the program will ask you to input the address interactively.

`/extras/setup` takes a required positional argument to specify the rack number
of the boot server.  For example, if the boot server is in the rack number 1,
run the command as follows:

```console
$ sudo env http_proxy=http://... /extras/setup 1
```

What it does are described below.

### Networking

It assigns link-local addresses to two physical NICs.
The link local addresses are used to communicate with Top-of-Rack (ToR) switches.
BIRD is configured to exchange routes with the ToR switches using BGP.

Additionally, it adds three virtual NICs to advertise routable addresses of the boot server.
Their names are `node0`, `bastion` and `boot`.

### NTP

It configures chrony to synchronize system clock with NTP servers.

### Host name

It sets the hostname appropriately.

### Purge and install packages

It removes unnecessary packages such as `unattended-upgrades` and installs packages such as `jq`.

### Install Docker

It installs Docker and configures it to use the HTTP proxy.

### Create files in `/etc/neco` directory

It creates `rack`, `cluster`, and `sabakan_ipam.json`.
Read [files.md](files.md) for more details.

### Upgrade packages

It upgrades stale packages.

[BIRD]: https://bird.network.cz/
[chrony]: https://chrony.tuxfamily.org/
