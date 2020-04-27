# Makefile for cke

GOFLAGS = -mod=vendor
export GOFLAGS
ETCD_VERSION = 3.3.19

all: test

setup:
	curl -fsL https://github.com/etcd-io/etcd/releases/download/v$(ETCD_VERSION)/etcd-v$(ETCD_VERSION)-linux-amd64.tar.gz | sudo tar -xzf - --strip-components=1 -C /usr/local/bin etcd-v$(ETCD_VERSION)-linux-amd64/etcd etcd-v$(ETCD_VERSION)-linux-amd64/etcdctl

test:
	test -z "$$(gofmt -s -l . | grep -v '^vendor' | tee /dev/stderr)"
	test -z "$$(golint $$(go list ./... | grep -v /vendor/) | grep -v '/mtest/.*: should not use dot imports' | tee /dev/stderr)"
	test -z "$$(nilerr ./... 2>&1 | tee /dev/stderr)"
	test -z "$$(custom-checker -restrictpkg.packages=html/template,log ./... 2>&1 | tee /dev/stderr)"
	ineffassign .
	go install ./pkg/...
	go test -race -v ./...
	go vet ./...

static:
	go generate ./static
	git add ./static/resources.go

mod:
	go mod tidy
	go mod vendor
	git add -f vendor
	git add go.mod

.PHONY:	all setup test static mod
