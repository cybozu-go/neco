neco
====

`neco` is the integrated tool to install/update miscellaneous programs
running on boot servers and worker nodes.

Synopsis
--------

* `neco setup LRN [LRN ...]`

    Install and setup etcd cluster as well as Vault using given boot servers.
    `LRN` is the logical rack number of the boot server.  At least 3 LRNs
    should be specified.

    This command should be invoked at once on all boot servers specified by LRN.

* `neco install NAME`

    Install a program in a boot server.  This command normally does not
    initialize data.

* `neco init NAME`

    Initialize data for a program.  This command should not be executed
    more than once.

* `neco update-all`

    Update all installed programs in a boot server.

    This command should be invoked at once on all boot servers.
    To gracefully upgrade programs, the command elects a leader server one after one.

* `neco update-saba [--sabakan-url URL]`

    Prepare assets and uploads them to [sabakan](https://github.com/cybozu-go/sabakan).

Etcd
----

`neco` command saves status and elects leaders using etcd.

The data structure in etcd is described in [etcd.md](etcd.md).
