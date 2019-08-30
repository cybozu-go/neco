Release procedure
=================

This document describes how to release a new version of sabakan.

Versioning
----------

Sabakan has two kind of versions: _API version_ and _schema version_.

API version should follow [semantic versioning 2.0.0][semver] to choose the new version number.
It is also used for archive versions.

Schema version is a positive integer.  It must be incremented when some data format has been changed.

Prepare change log entries
--------------------------

Add notable changes since the last release to [CHANGELOG.md](CHANGELOG.md).
It should look like:

```markdown
(snip)
## [Unreleased]

### Added
- Implement ... (#35)

### Changed
- Fix a bug in ... (#33)

### Removed
- Deprecated `-option` is removed ... (#39)

(snip)
```

Increment schema version
------------------------

When a backward-incompatible change is to be merged to `master`, the schema version must be incremented
and conversion from old schema need to be implemented.

1. Increment `SchemaVersion` in [version.go](./version.go) by 1.
2. Increment schema version at the top of [docs/schema.md](./docs/schema.md) by 1.
3. Add conversion method from old schema.  Example: [models/etcd/convert2.go](./models/etcd/convert2.go).
4. Call the conversion method from `driver.Upgrade` defined in [models/etcd/schema.go](./models/etcd/schema.go).

Bump version
------------

1. Determine a new API/program version number.  Let it write `$VERSION` as `VERSION=x.y.z`.
2. Checkout `master` branch.
3. Make a branch to release, for example by `git neco dev "$VERSION"`
4. Edit `CHANGELOG.md` for the new version ([example][]).
5. Update `Version` constant in [version.go](./version.go).
6. Commit the change and push it.

    ```console
    $ git commit -a -m "Bump version to $VERSION"
    $ git neco review
    ```

7. Merge this branch.
8. Checkout `master` branch.
9. Add a git tag, then push it.

    ```console
    $ git tag "v$VERSION"
    $ git push origin "v$VERSION"
    ```

Now the version is bumped up and the latest container image is uploaded to [quay.io](https://quay.io/cybozu/sabakan).

Publish GitHub release page
---------------------------

Go to https://github.com/cybozu-go/sabakan/releases and edit the tag.
Finally, press `Publish release` button.

[semver]: https://semver.org/spec/v2.0.0.html
[example]: https://github.com/cybozu-go/etcdpasswd/commit/77d95384ac6c97e7f48281eaf23cb94f68867f79
[CircleCI]: https://circleci.com/gh/cybozu-go/etcdpasswd
