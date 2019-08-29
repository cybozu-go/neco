# Change Log

All notable changes to this project will be documented in this file.
This project employs a versioning scheme described in [RELEASE.md](RELEASE.md#versioning).

## [Unreleased]

## [1.15.0-rc.4] - 2019-08-29

### Fixed
- Remove kube-proxy option(`--ipvs-strict-arp=true`) ([#223])

## [1.15.0-rc.3] - 2019-08-28

### Fixed
- Fix a bug that kubelet repeats rebooting (#221)

## [1.15.0-rc.2] - 2019-08-27

### Changed
- sabakan: update for gqlgen 0.9+ (#216)
- Update kubernetes to 1.15.3  (#219)
- Update etcd to 3.3.15  (#219)
- Update etcdutil to 1.3.3  (#219)
- Add readiness probes for cluster-dns and node-dns (#215)

### Fixed
- Fix a bug that multiple control planes can be selected from the same rack (#218)

## [1.15.0-rc.1] - 2019-08-19

### Changed
- Update kubernetes to 1.15.2  (#213)
- Update etcd to 3.3.14  (#213)
- Update CoreDNS to 0.7.5  (#213)
- Update Unbound to 1.9.2  (#213)
- Use `sigs.k8s.io/yaml` library (#212)
- Fix release document (#211)


## Ancient changes

* See [release-1.14/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.14/CHANGELOG.md) for changes in CKE 1.14.
* See [release-1.13/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.13/CHANGELOG.md) for changes in CKE 1.13.
* See [release-1.12/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.12/CHANGELOG.md) for changes in CKE 1.12.

[Unreleased]: https://github.com/cybozu-go/cke/compare/v1.15.0-rc.4...HEAD
[1.15.0-rc.4]: https://github.com/cybozu-go/cke/compare/v1.15.0-rc.3...v1.15.0-rc.4
[1.15.0-rc.3]: https://github.com/cybozu-go/cke/compare/v1.15.0-rc.2...v1.15.0-rc.3
[1.15.0-rc.2]: https://github.com/cybozu-go/cke/compare/v1.15.0-rc.1...v1.15.0-rc.2
[1.15.0-rc.1]: https://github.com/cybozu-go/cke/compare/v1.14.14...v1.15.0-rc.1
[#223]: https://github.com/cybozu-go/cke/pull/223
