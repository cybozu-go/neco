How to develop CKE
==================

## Go environment

Use Go 1.11.2 or higher.

CKE uses [Go modules](https://github.com/golang/go/wiki/Modules) to manage dependencies.
So you must either set `GO111MODULE=on` environment variable or checkout CKE out of `GOPATH`.

## Update dependencies

To update a dependency, just do:

```console
$ go get github.com/foo/bar@v1.2.3
```

After updating dependencies, run following commands to vendor dependencies:

```console
$ go mod tidy
$ go mod vendor
$ git add -f vendor
$ git add go.mod
$ git commit
```

### Maintenance branch

Each CKE release corresponds to a Kubernetes version.
For example, CKE 1.13.8 corresponds to Kubernetes 1.13.4.

When the `master` branch of CKE is updated for a new Kubernetes minor version (e.g. 1.14),
we should keep a maintenance branch for old (e.g. 1.13) Kubernetes.

Run following commands to create such a branch:

```console
$ git fetch origin
$ git checkout -b release-1.13 origin/master
$ git push -u origin release-1.13
```

### Update `k8s.io` modules

CKE uses `k8s.io/client-go`.

Modules under `k8s.io` are compatible with Go modules.
Therefore, when `k8s.io/client-go` is updated as follows, dependent modules are also updated.

```console
$ go get k8s.io/client-go@kubernetes-1.15.2

$ go mod tidy
$ go mod vendor
$ git add -f vendor
$ git add go.mod
$ git commit
```

### Update the Kubernetes resource definitions embedded in CKE.

The Kubernetes resource definitions embedded in CKE is defined in `./static/resource.go`.
This needs to be updated by `make static` whenever `images.go` updates.
