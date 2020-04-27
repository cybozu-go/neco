Release procedure
=================

This document describes how to release a new version of cke.

Versioning
----------

Given a version number MAJOR.MINOR.PATCH.
The MAJOR and MINOR version matches that of Kubernetes.
The patch version is increased with CKE update.

Maintain old k8s version
------------------------

If kubernetes MINOR version supported by CKE is updated, create a new branch `release-X.Y`
where `X` and `Y` are MAJOR and MINOR version of the latest release of CKE.

For example, if the last release of CKE was tagged as `v1.12.3` and want to start
development for Kubernetes 1.13 on master, create `release-1.12` branch as follows:

```console
$ git checkout -b release-1.12 v1.12.3
$ git push origin -u release-1.12:release-1.12
```

Remove old changes from `CHANGELOG.md` of master branch.
The `CHANGELOG.md` of master branch should only describe changes related to the latest release-X.Y.
Changes in the old versions are described in each branch's CHANGELOG.md.

`release-*` branches are protected from removal and force push.

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
3. Make a branch by `git neco dev "bump-$VERSION"`
4. Update `version.go`.
5. Edit `CHANGELOG.md` for the new version ([example][]).
6. Commit the change and create a pull request.

    ```console
    $ git commit -a -m "Bump version to $VERSION"
    $ git neco review
    ```
7. Make sure that Sonobuoy test has been passed when updating to `x.y.0` and its RC.
8. Merge the pull request.
9. Make a tag and push it.

    ```console
    $ git checkout master
    $ git pull
    $ git tag "v$VERSION"
    $ git push origin "v$VERSION"
    ```

Then CircleCI automatically builds and pushes the tagged container image to [quay.io](https://quay.io/cybozu/cke).

CircleCI also creates a GitHub release automatically after running [sonobuoy](./sonobuoy) tests.
So, **DO NOT MANUALLY CREATE GITHUB RELEASES**.  The test results will be attached to the GitHub
release that can be submitted to [cncf/k8s-conformance](https://github.com/cncf/k8s-conformance).

[example]: https://github.com/cybozu-go/etcdpasswd/commit/77d95384ac6c97e7f48281eaf23cb94f68867f79
