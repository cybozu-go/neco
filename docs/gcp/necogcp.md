necogcp
=======

`necogcp` is comamnd line tool for GCP provisioning.

Synopsis
--------

### GCE instance management

* `necogcp create [vmx-enabled|host-vm]`

    Create instance or instance image.
    If `vmx-enabled` is specified, it builds `vmx-enabled` image.
    If `host-vm` is specified, it launches `host-vm` instance using `vmx-enabled` image.
    If target image or instance already exists, it is re-created.

* `necogcp delete [vmx-enabled|host-vm]`

    Delete instance or instance image.
    If `vmx-enabled` is specified, it deletes `vmx-enabled` image.
    If `host-vm` is specified, it deletes `host-vm` instance.

### Miscellaneous

* `necogcp completion`

    Dump bash completion rules for `necogcp` command.

Flags
-----

Flag            | Default value    | Description
--------------- | ---------------- | -----------
`--config`      | `$HOME/.necogcp.yml` | [Viper configuration file](https://github.com/spf13/viper#reading-config-files).

Configuration file
------------------

See [config.md](config.md)
