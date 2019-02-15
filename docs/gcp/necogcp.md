necogcp
=======

`necogcp` is comamnd line tool for GCP provisioning.

Synopsis
--------

### GCE instance management on developer's project

* `necogcp create-image`

    It builds `vmx-enabled` image.
    If `vmx-enabled` image already exists, it is re-created.

* `necogcp create-instance`

    It launches `host-vm` instance using `vmx-enabled` image.
    If `host-vm` instance already exists, it is re-created.
    
* `necogcp delete-image`

    It deletes `vmx-enabled` image.

* `necogcp delete-instance`

    It deletes `host-vm` instance.

### GCE instance management on neco-test project

* `necogcp neco-test create-image`

    It builds `vmx-enabled` image on neco-test.
    If `vmx-enabled` image already exists, it is re-created.

* `necogcp neco-test delete-image`

    It deletes `vmx-enabled` image on neco-test.

* `necogcp neco-test extend INSTANCE_NAME`

    It extends 1 hours given instance on the neco-test project to prevent deleted by GAE app.

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
