GCP
===

Requirements
------------

- [Cloud SDK](https://cloud.google.com/sdk/)

Overview
--------

![workflow](http://www.plantuml.com/plantuml/svg/ZP1DJWGX58JtdABISQ6BngWxJCmOntW23u3mGdVaXpGmqTjBbEb5_vXPlbVKzuewf2odN9nbKkKmeRlZE5AquHjlpw-GCNPdvPxZPIAP2IVtrPEN7vOqHYSDpMyUEDuJOGWfzoU7qLUyLrKlYPIiIhVXPM9rLA3ldt3TfXi8N45f61Llw_m89py9-jV2n3_16cUr_oDwKg4YeLPk9dP-8i3vnS1cteMrT8jrOnbbGz5GmSLIMIaDCs_hFETR975kPgcWfD4Rh41ixdm_sv4iLQJCyACPCZkGryNP7m00)

Developer's development environment is deployed by GCE.
The instance name is called `host-vm` which is based on `vmx-enabled` instance image.
Developer can create custom `vmx-enabled` image with `$HOME/.neco-gcp.yml`.

GAE app
-------

**To prevent over cost of GCP billing, You have to deploy GAE app in advance.**

GAE app on your GCP project does:

- Stop given instances at night.
- Delete `host-vm` instance and other all instances at night.

Usage
-----

### Setup your configuration file

First, download a credential to access your GCP account by following steps:

1. Configure `$HOME/.necogcp.yml` using [necogcp-example.yml](necogcp-example.yml).
1. Run `gcloud auth login` with your GCP project.
1. Edit `$HOME/.necogcp.yml`. See [config.md](config.md)

### Deploy GAE app for your project

You can skip this step if the GAE app is up to date.

```console
make -f Makefile.gcp create
```
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

**`host-vm` instance is deleted by GAE app every evening, You have to run above step every day.**

Please create `vmx-enabled` image in advance with above step.

```console
necogcp create-instance
```

If you want to update your existing image, re-run this command.


For `neco-test` GCP project
---------------------------

See [neco-test.md](neco-test.md)
