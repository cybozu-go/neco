GCP
===

Requirements
------------

- [Cloud SDK](https://cloud.google.com/sdk/)

Overview
--------

![workflow](http://www.plantuml.com/plantuml/svg/bT1DIiKm50NWULPnA0vqq80woq2U-CGLk82ONj92Va9pyz3TMwKDs1_X3NjFy-NadZBhaOjtGXkE8vfxYJCb5x_mzlmAdLAp90yIBoDf9bpyPqj1VpZgC7XjgVxpAF6U2NtCt5uyXZ3nmKnvoMJqb3GxPuNQNAhGja-udR_ke723G2PQatu6CBz5XFgdqqnivdyn4tqgJ30RHQY3noX8UILSaIDqQRiQpwOrBfQkaZdtrTkh8UMUf9PDhHEKF0IB3IJg-e-xdSaH4pGZ3BPdSQBG5U--0G00)

Developer's development environment is deployed by GCE.
The instance name is called `host-vm` which is based on `vmx-enabled` instance image.
Developer can create custom `vmx-enabled` image with `$HOME/.neco-gcp.yml`.

GAE app
-------

**To prevent over cost of GCP billing, You have to deploy GAE app in advance.**

GAE app on your GCP project does:

- Stop all instances at night.
- Delete `host-vm` instance and given instances at night.

Usage
-----

### Setup your configuration file

First, download a credential to access your GCP account by following steps:

1. Configure `$HOME/.necogcp.yml` using [necogcp-example.yml](necogcp-example.yml).
1. Run `gcloud auth login` with your GCP project.
1. Edit `$HOME/.necogcp.yml`. See [config.md](config.md)

### Deploy GAE app for your project

You can skip this step if it's already deployed or is up to date.

```console
cd gcp/app
make CONFIG=$HOME/.necogcp.yml deploy
```

### Create `vmx-enabled` instance image for your project

```console
necogcp compute create vmx-enabled
```

If you want to update your existing image, run `necogcp compute create -u vmx-enabled`.

### Use `host-vm` instance for your project

Please create `vmx-enabled` image in advance with above step.

```console
necogcp compute create host-vm
```

If you want to update your existing image, run `necogcp compute create -u host-vm`.

After login, please make sure vmx is enabled by `grep -cw vmx /proc/cpuinfo`.

**`host-vm` instance is deleted by GAE app every evening, You have to run above step every day.**

For `neco-test` GCP project
---------------------------

See [neco-test.md](neco-test.md)
