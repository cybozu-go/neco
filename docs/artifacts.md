Artifacts
=========

[Artifacts](../artifacts.go) is the collection of components tested in [dctest](../dctest/), which consists of container images, debian package, and Flatcar image.

There is one artifacts file.  When generating neco binaries and the neco deb package, `artifacts.go` is used by default.

- [artifacts.go](../artifacts.go)

    This file describes artifacts to be tested in development.
    It should be updated with `generate-artifacts` command.
    You may edit the file **ONLY IF** you would like to test different combination of components.

## How to handle prerelease versions

As described above, you should not edit artifacts files manually in general.
One of the exceptions is to use an RC version of a component, e.g. cke 1.15.0-rc1,
to prepare for the update of the component.
A component with prerelease version information, i.e. x.y.z-\<prerelease\>,
does not get included in artifacts files by `generate-artifacts`.
So if you want to use such a component, include it manually.
Though you can edit `artifacts.go` in this case, you must not merge it into
the main branch anyway.
Instead, after you confirm that neco can accept a component of x.y.z-\<prerelease\>,
release the component as x.y.z and include it in artifacts files by `generate-artifacts`.

## How to ignore latest versions

By using `artifacts_ignore.yaml`, the latest versions of components can be ignored.
Use this, for example, if you have a bug in the latest versions and do not want to include it in `artifacts.go`.

```yaml
images:
- repository: ghcr.io/cybozu-go/cke
  versions: ["1.2.3", "1.2.4", "1.2.5"]
- repository: ghcr.io/cybozu/etcd
  versions: ["1.2.3.4"]
debs:
- name: etcdpasswd
  versions: ["v1.2.3"]
osImage:
- channel: stable
  versions: ["2247.5.0"]
```
