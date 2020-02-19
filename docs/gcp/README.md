GCP
===

Requirements
------------

- [Cloud SDK](https://cloud.google.com/sdk/)

Overview
--------

![workflow](http://www.plantuml.com/plantuml/svg/ZT0xoiCm40JWNgSOp5_yAIH8IXN1a3Fa0CfwaGrw64ioENvSH2dy26btD9-MRLCsKoxU2KCvJcZE2hU9JMRC_Yavc8VZ3eCtbflwvg9mJum-fYndZo4iIA0bBud9B4cpqnNw2wqXvHN_c_a96dy8JD7I2DgqXJxOHKEfNy5QFiBgTglnsxBaOkb0qOybCrBgFzxUzqhjIRfUPfsWf25OR23HSkYAToy0)

Developer's development environment is deployed by GCE.
The instance name is called `host-vm` which is based on `vmx-enabled` instance image.
Developer can create custom `vmx-enabled` image with `$HOME/.neco-gcp.yml`.

Usage
-----

### Setup your configuration file

First, download a credential to access your GCP account by following steps:

1. Configure `$HOME/.necogcp.yml` using [necogcp-example.yml](necogcp-example.yml).
1. Run `gcloud auth login` with your GCP project.
1. Edit `$HOME/.necogcp.yml`. See [config.md](config.md)

### Install necogcp command

necogcp command is used for creating a VM image, creating a VM instance, and so on.

```console
make necogcp
```

### Create `vmx-enabled` instance image for your project

```console
necogcp create-image
```

If you want to update your existing image, re-run this command.

### Use `host-vm` instance for your project

**`host-vm` instance is deleted by `at` job which is created by `necogcp create-instance`**

Please create `vmx-enabled` image in advance with above step.

```console
necogcp create-instance
```

If you want to update your existing image, re-run this command.

### Extend time to delete `host-vm` automatically

`host-vm` created by `necogcp create-instance` has a `at` job which deletes `host-vm` instance at 21:00 JST by default.
If you want to extend the time, modify the `at` job owned by the root user.
Also, if you want to change the deletion time of your `host-vm` instance permanently, modify the `shutdown-at` field in [this example](necogcp-example.yml). This change affects only when executing `necogcp create-instance`.


For `neco-test` GCP project
---------------------------

See [neco-test.md](neco-test.md)
