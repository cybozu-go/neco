# `download-binaries`

## What's this?

We often forgot upgrading of the CLI tools using for operations.
To solve this problem we decided to automate the process to upload the binaries on operation environments.

`download-binaries` fetches the following binaries:

#### `teleport`

Download the binary from [the download page](https://gravitational.com/teleport/download/).

The version to use is written on [artifacts.go](https://github.com/cybozu-go/neco/blob/master/artifacts.go).

#### `kubectl`

Download the binary from [the bucket](https://storage.googleapis.com/kubernetes-release/release).

The version is decided by the following process:

1. Check the CKE version written in artifacts.go. We can know the major version and minor version of Neco's Kubernetes.
1. Check latest patch version from GitHub release in k/k.

#### `argocd`

Download the binary from [the release page](https://github.com/argoproj/argo-cd/releases).

The version to use is written in [neco-apps/argocd/base/upstream/install.yaml](https://github.com/cybozu-go/neco-apps/blob/release/argocd/base/upstream/install.yaml)

However, ArgoCD repository does not distribute Windows binaries now, so `download-binaries` skips to download `argocd`. This document is written at 2020-02-03.
