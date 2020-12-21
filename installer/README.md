This directory contains tools to create custom Ubuntu installers for
boot servers.

## Prerequisites

The Makefile assumes to be run on a recent Ubuntu or Debian OS.
To test built images, QEMU/KVM is used.

## Build

1. Prepare `cluster.json` file in this directory.

    The contents should be a JSON array of objects with these fields:

    | Name              | Type   | Description                              |
    | ----------------- | ------ | ---------------------------------------- |
    | `name`            | string | Cluster name                             |
    | `bastion_network` | string | IPv4 address of the bastion network      |
    | `bmc_network`     | CIDR   | IPv4 network address in CIDR for the BMC |
    | `ntp_servers`     | array  | List of NTP server addresses             |

    [`cluster.json.example`](./cluster.json.example) is an example of this file.

2. Run `make` to see available build options.
3. Run `make setup`.  This is a one-time procedure.
4. Run `make all` to build everything.

- `build/cybozu-ubuntu-20.04-live-server-amd64.iso` is the custom ISO installer.
- `build/cybozu-ubuntu-20.04-server-cloudimg-amd64.img` is the custom cloud image.

## Test

Built images can be tested with `make preview-iso` and `make preview-cloud`.

They require a Linux host that can run QEMU/KVM.  If your host itself is a
virtual machine, you need to enable nested virtualization.
For Hyper-V, run the following command in *administrator's* PowerShell console
after shutting down the virtual machine.

```console
$ Set-VMProcessor -VMName <VMName> -ExposeVirtualizationExtensions $true
```

To run QEMU/KVM w/o root privileges, add `kvm` group to your account.
You need to logout and login to gain the group privilege.

```console
$ sudo adduser $USER kvm
$ exit
```

They also require X Window System.  [MobaXterm](https://mobaxterm.mobatek.net/)
is quite handy to run X on Windows.
