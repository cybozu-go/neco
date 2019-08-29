# Makefile for sabakan

GO_FILES=$(shell find -name '*.go' -not -name '*_test.go')
BUILT_TARGET=sabakan sabactl sabakan-cryptsetup
ETCD_DIR = /tmp/neco-etcd

# for Go
GOFLAGS = -mod=vendor
export GOFLAGS

all: test

build: $(BUILT_TARGET)
$(BUILT_TARGET): $(GO_FILES)
	go build ./pkg/$@

start-etcd:
	systemd-run --user --unit neco-etcd.service etcd --data-dir $(ETCD_DIR)

stop-etcd:
	systemctl --user stop neco-etcd.service

test: build
	test -z "$$(gofmt -s -l . | grep -v '^vendor' | tee /dev/stderr)"
	test -z "$$(golint $$(go list ./... | grep -v /vendor/) | grep -v '/mtest/.*: should not use dot imports' | tee /dev/stderr)"
	ineffassign .
	go test -race -v ./...
	go vet ./...

e2e: build
	RUN_E2E=1 go test -v -count=1 ./e2e

mod:
	go mod tidy
	go mod vendor
	git add -f vendor
	git add go.mod

clean:
	rm -f $(BUILT_TARGET)

.PHONY: all build start-etcd stop-etcd test e2e mod clean
