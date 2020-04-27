How to develop CKE
==================

## Go environment

Use Go 1.13.8 or higher.

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
For example, CKE 1.16.x corresponds to Kubernetes 1.16.x.

We should keep a maintenance branch for old (e.g. 1.16) Kubernetes:

When the `master` branch of CKE is updated for a new Kubernetes minor version (e.g. 1.17),
we should keep a maintenance branch for old (e.g. 1.16) Kubernetes.

Run the following commands to create such a branch:

```console
$ git fetch origin
$ git checkout -b release-1.16 origin/master
$ git push -u origin release-1.16
```

When vulnerabilities or critical issues are found in the master branch, 
we should backport the fixes to an older branch.

Run following commands to backport:

```
$ git checkout release-1.16
$ git cherry-pick <commit from master>
```

Then, release it. 
https://github.com/cybozu-go/cke/blob/master/RELEASE.md

### Update `k8s.io` modules

CKE uses `k8s.io/client-go`.

Modules under `k8s.io` are compatible with Go modules.
Therefore, when `k8s.io/client-go` is updated as follows, dependent modules are also updated.

```console
$ go get k8s.io/client-go@0.17.4

$ go mod tidy
$ go mod vendor
$ git add -f vendor
$ git add go.mod
$ git commit
```

### Update the Kubernetes resource definitions embedded in CKE.

The Kubernetes resource definitions embedded in CKE is defined in `./static/resource.go`.
This needs to be updated by `make static` whenever `images.go` updates.
