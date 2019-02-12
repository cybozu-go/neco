# Makefile for neco

SUDO = sudo
FAKEROOT = fakeroot
ETCD_DIR = /tmp/neco-etcd
TAGS =

### for Go
GOFLAGS = -mod=vendor
GOTAGS = $(TAGS) containers_image_openpgp containers_image_ostree_stub
export GOFLAGS

### for debian package
PACKAGES := fakeroot btrfs-tools pkg-config libdevmapper-dev
WORKDIR := $(CURDIR)/work
CONTROL := $(WORKDIR)/DEBIAN/control
DOCDIR := $(WORKDIR)/usr/share/doc/neco
NODE_EXPORTER_DOCDIR := $(WORKDIR)/usr/share/doc/node_exporter
BINDIR := $(WORKDIR)/usr/bin
SBINDIR := $(WORKDIR)/usr/sbin
SHAREDIR := $(WORKDIR)/usr/share/neco
VERSION = 0.0.1-master
DEST = .
DEB = neco_$(VERSION)_amd64.deb
BIN_PKGS = ./pkg/neco
SBIN_PKGS = ./pkg/neco-updater ./pkg/neco-worker ./pkg/sabakan-serf-handler
NODE_EXPORTER_VERSION = 0.17.0
NODE_EXPORTER_PATH = $(GOPATH)/src/github.com/prometheus/node_exporter

all:
	@echo "Specify one of these targets:"
	@echo
	@echo "    start-etcd  - run etcd on localhost."
	@echo "    stop-etcd   - stop etcd."
	@echo "    test        - run single host tests."
	@echo "    mod         - update and vendor Go modules."
	@echo "    deb         - build Debian package."
	@echo "    setup       - install dependencies."

start-etcd:
	systemd-run --user --unit neco-etcd.service etcd --data-dir $(ETCD_DIR)

stop-etcd:
	systemctl --user stop neco-etcd.service

test:
	test -z "$$(gofmt -s -l . | grep -v '^vendor' | tee /dev/stderr)"
	test -z "$$(golint $$(go list -tags='$(GOTAGS)' ./... | grep -v /vendor/) | grep -v '/dctest/.*: should not use dot imports' | tee /dev/stderr)"
	go build -tags='$(GOTAGS)' ./...
	go test -tags='$(GOTAGS)' -race -v ./...
	RUN_COMPACTION_TEST=yes go test -tags='$(GOTAGS)' -race -v -run=TestEtcdCompaction ./worker/
	go vet -tags='$(GOTAGS)' ./...

mod:
	go mod tidy
	go mod vendor
	git add -f vendor
	git add go.mod go.sum

node_exporter:
	GO111MODULE=off go get github.com/prometheus/node_exporter
	cd $(NODE_EXPORTER_PATH) && \
		git checkout v$(NODE_EXPORTER_VERSION) && \
		make PREFIX=$(GOPATH)/src/github.com/cybozu-go/neco build

deb: $(DEB)

$(DEB): node_exporter
	rm -rf $(WORKDIR)
	cp -r debian $(WORKDIR)
	sed 's/@VERSION@/$(patsubst v%,%,$(VERSION))/' debian/DEBIAN/control > $(CONTROL)
	mkdir -p $(BINDIR)
	GOBIN=$(BINDIR) go install -tags='$(GOTAGS)' $(BIN_PKGS)
	mkdir -p $(SBINDIR)
	GOBIN=$(SBINDIR) go install -tags='$(GOTAGS)' $(SBIN_PKGS)
	mkdir -p $(NODE_EXPORTER_DOCDIR)
	cp $(NODE_EXPORTER_PATH)/LICENSE $(NODE_EXPORTER_PATH)/NOTICE $(NODE_EXPORTER_DOCDIR)
	mv $(GOPATH)/src/github.com/cybozu-go/neco/node_exporter $(SBINDIR)
	mkdir -p $(SHAREDIR)
	cp etc/* $(SHAREDIR)
	cp -a ignitions $(SHAREDIR)
	mkdir -p $(DOCDIR)
	cp README.md LICENSE $(DOCDIR)
	chmod -R g-w $(WORKDIR)
	$(FAKEROOT) dpkg-deb --build $(WORKDIR) $(DEST)
	rm -rf $(WORKDIR)

setup:
	GO111MODULE=off go get -u golang.org/x/lint/golint
	$(SUDO) apt-get update
	$(SUDO) apt-get -y install --no-install-recommends $(PACKAGES)

clean:
	rm -rf $(ETCD_DIR) $(WORKDIR) $(DEB)

.PHONY:	all start-etcd stop-etcd test mod deb setup clean
