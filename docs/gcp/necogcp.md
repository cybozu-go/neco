necogcp
=======

`necogcp` is a command-line tool for GCP provisioning.

Synopsis
--------

### GCE instance management on developer's project

* `necogcp create-image`

    Build `vmx-enabled` image.
    If `vmx-enabled` image already exists, it is re-created.

* `necogcp create-instance`

    Launch `host-vm` instance using `vmx-enabled` image.
    If `host-vm` instance already exists, it is re-created.

* `necogcp delete-image`

    Delete `vmx-enabled` image.

* `necogcp delete-instance`

    Delete `host-vm` instance.

* `necogcp setup-instance`

    Setup `host-vm` or `vmx-enabled` instance. It can run on only them.

### GCE instance management on neco-test project

* `necogcp neco-test create-image`

    Build `vmx-enabled` image on neco-test.
    If `vmx-enabled` image already exists, it is re-created.

* `necogcp neco-test delete-image`

    Delete `vmx-enabled` image on neco-test.

* `necogcp neco-test extend INSTANCE_NAME`

    Extend 2 hours given instance on the neco-test project to prevent deleted by GAE app.

### Miscellaneous

* `necogcp completion`

    Dump bash completion rules for `necogcp` command.

Flags
-----

| Flag       | Default value        | Description                                                                      |
| ---------- | -------------------- | -------------------------------------------------------------------------------- |
| `--config` | `$HOME/.necogcp.yml` | [Viper configuration file](https://github.com/spf13/viper#reading-config-files). |

Configuration file
------------------

See [config.md](config.md)
