Data Center Test (dctest)
=========================

[dctest](../dctest/) directory contains test suites to run integration
tests in a virtual data center environment.

Generating deb Package
----------------------

Dctest uses a generated deb package to install Neco.
Details of artifacts definition used in generating the deb package are
described in [Artifacts](artifacts.md).

Type of Test Suites
-------------------

There are three types of test suites.

1. bootstrap

    This suite tests initial setup of Neco.  This does not include
    upgrading test nor boot server node joining/leaving test.

    This suite installs Neco with the generated deb package if `DATACENTER`
    is not specified.

    If `DATACENTER` is specified, this test suite is invoked to prepare the base
    of upgrading test.  The Neco deb package used for `DATACENTER`, staging or
    production, is downloaded from GitHub releases.

2. upgrade

    This suite tests upgrade of Neco. This suite is not self-contained. It depends on bootstrap.

    Before running upgrade test suite, bootstrap test with an old Neco package and old procedures must be executed.
    This old package is downloaded from GitHub releases, and the old procedures get checked-out from git repository.
    Upgrade test suite then upgrades Neco with the generated deb package,
    which is versioned as `9999.12.31-99999`.

3. functions

    This suite tests a full set of functions of Neco in a single version,
    i.e. this consists of initial setup of Neco and joining/leaving of
    a boot server node.

    This suite installs Neco with the generated deb package.

4. reboot

    This suite tests disaster recovery scenario.
    It includes `functions` suite, so this suite takes more time than `functions` or `bootstrap`. 

Each test suite has an entry point of test as `<suite>/suite_test.go`.

### Base of upgrading test

As described above, upgrading test first installs Neco with an uploaded deb
package.  This is to reproduce a real-world deployed data center environment
as the base of upgrade.

There are two types of data center environments to be reproduced: `production`
and `staging`.  Upgrading test decides which version of a deb package to use
by the data center environment with the same logic as
[automatic update](update.md#tag-name-and-release-flow).

The test environment must keep backward compatibility to run old Neco packages.

Synopsis
--------

[`Makefile`](../Makefile) setup virtual data center environment and runs dctest.

* `make setup`

    Install dctest required components.

* `make clean`

    Delete generated files in `output/` directory.

* `make placemat`

    Run `placemat` in background by systemd-run to start virtual machines.

* `make stop`

    Stop `placemat`.

* `make test`

    Run dctest on a running `placemat`.  This does not contol `placemat` by itself.

Options
-------

### `SUITE`

You can choose the type of test suite by specifying `SUITE` make variable.
The value can be `bootstrap` (default), `upgrade`, `functions`, or `reboot`.

`make test` accepts this variable.

### `DATACENTER`

When building the base of upgrading test with `SUITE=bootstrap`,
you can choose reproduced environment by specifying `DATACENTER` make
variable.
The value can be `staging` or `production`.

This variable makes sense only when `SUITE=bootstrap` is specified.

`make test` accepts this variable.

### `TAGS`

This variable is not currently in use.

`make test` accepts this variable.

### `MENU_ARG`

You can configure the virtual data center constructed by `placemat` via `MENU_ARG` file.
Read a `placemat-menu` [document](./placemat-menu.md) for more detail.
If you change the parameter from previously used one, you should run `make clean` to apply the configuration properly.

`make placemat` accepts this variable.

### `github-token` file

`neco-updater` watches GitHub release without authentication by default. `neco-worker` also watches it to download debian packages.
It would receive rate limits of the GitHub API. dctest runs `neco config set github-token` automatically when `github-token` file exists.
