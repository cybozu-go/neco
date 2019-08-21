Release procedure
=================

This document describes how to release a new version of etcdutil.

Versioning
----------

Follow [semantic versioning 2.0.0][semver] to choose the new version number.

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

Bump version
------------

1. Determine a new version number.  Let it write `$VERSION` as `VERSION=x.y.z`.
2. Checkout `master` branch.
3. Make a branch to release, for example by `git neco dev "$VERSION"`
4. Edit `CHANGELOG.md` for the new version ([example][]).
5. Commit the change and push it.

    ```console
    $ git commit -a -m "Bump version to $VERSION"
    $ git neco review
    ```

6. Merge this branch.
7. Checkout `master` branch.
8. Add a git tag, then push it.

    ```console
    $ git tag "v$VERSION"
    $ git push origin "v$VERSION"
    ```

Publish GitHub release page
---------------------------

Go to https://github.com/cybozu-go/sabakan/releases and edit the tag.
Finally, press `Publish release` button.

[semver]: https://semver.org/spec/v2.0.0.html
[example]: https://github.com/cybozu-go/etcdpasswd/commit/77d95384ac6c97e7f48281eaf23cb94f68867f79
