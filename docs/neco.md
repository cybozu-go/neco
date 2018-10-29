neco
====

`neco` is an interactive tool for administrators.

It installs/updates miscellaneous programs as well as maintaining etcd database.

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

* `neco add`

    Add this server as a new boot server.

* `neco remove SERVER`

    Unregister `SERVER` from etcd.
