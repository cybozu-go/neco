necogcp
=======

`necogcp` is comamnd line tool for GCP provisioning.

Synopsis
--------

### GCE instance management

* `necogcp create [-u] [vmx-enabled|host-vm]`

    Create instance or instance image.
    If `vmx-enabled` is specified, it builds `vmx-enabled` image.
    If `host-vm` is specified, it launches `host-vm` instance using `vmx-enabled` image.
    `-u` option re-creates target image or instance.

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
