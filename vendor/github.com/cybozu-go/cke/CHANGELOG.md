# Change Log

All notable changes to this project will be documented in this file.
This project employs a versioning scheme described in [RELEASE.md](RELEASE.md#versioning).

## [Unreleased]

## [1.14.13] - 2019-08-01

### Changed
- Fix a bug that prevents Node resource creation in some cases (#203)
- Maintain default/kubernetes Endpoints by CKE itself (#204)

## [1.14.12] - 2019-07-19

### Added

- sabakan: weighted selection of node roles (#200)

## [1.14.11] - 2019-07-16

### Added

- Labels and taints for control plane nodes (#199)

## [1.14.10] - 2019-07-12

### Changed

- Fix bug on getting status.Scheduler, again (#198)

## [1.14.9] - 2019-07-11

### Changed

- Rename to recommended label keys (#197).

## [1.14.8] - 2019-07-09

### Added

- Invoke vault tidy periodically (#196).

### Fixed

- log: be silent when checking scheduler status (#195).
- mtest: use docker instead of podman (#194).

## [1.14.7] - 2019-06-28

### Fixed

- Fix bug on getting status.Scheduler (#193)

## [1.14.6] - 2019-06-27

### Added

- Add scheduler extender configurations in cluster.yml (#191, #192)

## [1.14.5] - 2019-06-14

### Added

- Add `ckecli sabakan enable` (#190).

## [1.14.4] - 2019-06-05

### Changed

- Fix `ckecli vault init` for newer vault API, and test re-init by mtest (#187).

## [1.14.3] - 2019-06-04

### Action required

- Updating of the existing installation requires re-invocation of `ckecil vault init`
    to add a new CA to Vault.

### Added

- Enable [API aggregation layer](https://kubernetes.io/docs/tasks/access-kubernetes-api/configure-aggregation-layer/) (#186).

## [1.14.2] - 2019-06-04

### Changed

- Fix a bug to stop CKE when it fails to connect to vault (#185).

## [1.14.1] - 2019-05-28

### Added
- Add `etcd-rivers` as a reverse proxy for k8s etcd (#181).
- Add `--follow` option to `ckecli history` (#180).

### Changed
- Apply nilerr and restrictpkg to test (#176).
- Add a cke container test to mtest (#175).
- Refine the output of `ckecli history` (#170).
- Fix handling etcd API `WithLimit` (#177).
- Fix dockerfile for podman v1.3.2-dev (#182).

## [1.14.0] - 2019-04-22

No user-visible changes since RC 1.

## [1.14.0-rc1] - 2019-04-19

### Changed
- Update kubernetes to 1.14.1
- Update CoreDNS to 1.5.0
- Update CNI plugins to 0.7.5

## [1.13.18] - 2019-04-17

### Added
- Build docker image by this repository instead of github.com/cybozu/neco-containers (#162)

### Changed
- Fix docker image bug (#163, #168).
- Run kubelet with docker option `--tmpfs=/tmp` (#167).
- Improve mtest environment and CI (#161, #164, #166).
- Update document (#165).

## [1.13.17] - 2019-03-30

### Added
- Kubernetes Secrets are encrypted at rest in etcd (#160).

## [1.13.16] - 2019-03-27

### Changed
- Fix infinite loop when image is updated for a system resource (#159).

## [1.13.15] - 2019-03-26

### Changed
- Upgrade CoreDNS to 1.4.0 (#158).

## [1.13.14] - 2019-03-26

### Changed
- Fix API versions to be used for resources (#157).

## [1.13.13] - 2019-03-25

### Changed
- Fix API versions to be used for resources (#154).
- Recreate user-defined resource when applying patches fail (#155).

## [1.13.12] - 2019-03-20

### Changed
- Avoid thundering herd problem (#153).

## [1.13.11] - 2019-03-19

### Changed
- Always patch existing resources to avoid update forbidden errors (#151).
- CKE waits enough number of nodes to be ready before creating resources (#152).

## [1.13.10] - 2019-03-18

### Changed
- CKE is ready for enabling PodSecurityPolicy (#149).

## [1.13.9] - 2019-03-15

### Added
- [User-defined resources](docs/user-resources.md) (#145).

### Changed
- Enable [NodeRestriction admission controller](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#noderestriction) (#148).

## [1.13.8] - 2019-03-08

### Changed
- Correct kube-proxy flags to handle load balancers with `externalTrafficPolicy=Local` (#139).
- Retry image pulling to be more robust (#140).

## [1.13.7] - 2019-03-07

### Added
- CNI configuration can be specified in `cluster.yml` (#136).

### Changed
- Fix a bug that prevents kubelet to be restarted cleanly (#138).

## [1.13.6] - 2019-03-07

### Changed
- Update Kubernetes to 1.13.4 (#137).
- Apply kube-proxy patch to fix kubernetes/kubernetes#72432 (#137).

## [1.13.5] - 2019-03-01

### Changed
- Remove the step to pull `pause` container image (#135).

## [1.13.4] - 2019-02-26

### Added
- Support remote runtime for kubernetes pod (#133).
- Support log rotation of remote runtime for kubelet configuration (#133).

## [1.13.3] - 2019-02-12

### Added
- Add audit log support (#130).

### Changed
- Fix removing node resources if hostname in cluster.yaml is specified (#129).

## [1.13.2] - 2019-02-07

### Added
- [FAQ](./docs/faq.md).

### Changed
- `ckecli ssh` does not look for the node in `cluster.yml` (#127).
- kubelet reports OS information correctly (#128).
- When kubelet restarts, OOM score adjustment did not work (#128).
- Specify rshared mount option instead of shared for /var/lib/kubelet (#128).

## [1.13.1] - 2019-02-06

### Changed
- Logs from Kubernetes programs (apiserver, kubelet, ...) and etcd are sent to journald (#126).

## [1.13.0] - 2019-01-25

### Changed
- Support for kubernetes 1.13 (#125).
- Update etcd to 3.3.11, CoreDNS to 1.3.1, unbound to 1.8.3.

## Ancient changes

See [CHANGELOG-1.12](./CHANGELOG-1.12.md).

[Unreleased]: https://github.com/cybozu-go/cke/compare/v1.14.13...HEAD
[1.14.13]: https://github.com/cybozu-go/cke/compare/v1.14.12...v1.14.13
[1.14.12]: https://github.com/cybozu-go/cke/compare/v1.14.11...v1.14.12
[1.14.11]: https://github.com/cybozu-go/cke/compare/v1.14.10...v1.14.11
[1.14.10]: https://github.com/cybozu-go/cke/compare/v1.14.9...v1.14.10
[1.14.9]: https://github.com/cybozu-go/cke/compare/v1.14.8...v1.14.9
[1.14.8]: https://github.com/cybozu-go/cke/compare/v1.14.7...v1.14.8
[1.14.7]: https://github.com/cybozu-go/cke/compare/v1.14.6...v1.14.7
[1.14.6]: https://github.com/cybozu-go/cke/compare/v1.14.5...v1.14.6
[1.14.5]: https://github.com/cybozu-go/cke/compare/v1.14.4...v1.14.5
[1.14.4]: https://github.com/cybozu-go/cke/compare/v1.14.3...v1.14.4
[1.14.3]: https://github.com/cybozu-go/cke/compare/v1.14.2...v1.14.3
[1.14.2]: https://github.com/cybozu-go/cke/compare/v1.14.1...v1.14.2
[1.14.1]: https://github.com/cybozu-go/cke/compare/v1.14.0...v1.14.1
[1.14.0]: https://github.com/cybozu-go/cke/compare/v1.14.0-rc1...v1.14.0
[1.14.0-rc1]: https://github.com/cybozu-go/cke/compare/v1.13.18...v1.14.0-rc1
[1.13.18]: https://github.com/cybozu-go/cke/compare/v1.13.17...v1.13.18
[1.13.17]: https://github.com/cybozu-go/cke/compare/v1.13.16...v1.13.17
[1.13.16]: https://github.com/cybozu-go/cke/compare/v1.13.15...v1.13.16
[1.13.15]: https://github.com/cybozu-go/cke/compare/v1.13.14...v1.13.15
[1.13.14]: https://github.com/cybozu-go/cke/compare/v1.13.13...v1.13.14
[1.13.13]: https://github.com/cybozu-go/cke/compare/v1.13.12...v1.13.13
[1.13.12]: https://github.com/cybozu-go/cke/compare/v1.13.11...v1.13.12
[1.13.11]: https://github.com/cybozu-go/cke/compare/v1.13.10...v1.13.11
[1.13.10]: https://github.com/cybozu-go/cke/compare/v1.13.9...v1.13.10
[1.13.9]: https://github.com/cybozu-go/cke/compare/v1.13.8...v1.13.9
[1.13.8]: https://github.com/cybozu-go/cke/compare/v1.13.7...v1.13.8
[1.13.7]: https://github.com/cybozu-go/cke/compare/v1.13.6...v1.13.7
[1.13.6]: https://github.com/cybozu-go/cke/compare/v1.13.5...v1.13.6
[1.13.5]: https://github.com/cybozu-go/cke/compare/v1.13.4...v1.13.5
[1.13.4]: https://github.com/cybozu-go/cke/compare/v1.13.3...v1.13.4
[1.13.3]: https://github.com/cybozu-go/cke/compare/v1.13.2...v1.13.3
[1.13.2]: https://github.com/cybozu-go/cke/compare/v1.13.1...v1.13.2
[1.13.1]: https://github.com/cybozu-go/cke/compare/v1.13.0...v1.13.1
[1.13.0]: https://github.com/cybozu-go/cke/compare/v1.12.0...v1.13.0
