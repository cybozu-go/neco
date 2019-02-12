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
DOCDIR := $(WORKDIR)/usr/share/doc
BINDIR := $(WORKDIR)/usr/bin
SBINDIR := $(WORKDIR)/usr/sbin
SHAREDIR := $(WORKDIR)/usr/share/neco
VERSION = 0.0.1-master
DEST = .
DEB = neco_$(VERSION)_amd64.deb
BIN_PKGS = ./pkg/neco
SBIN_PKGS = ./pkg/neco-updater ./pkg/neco-worker ./pkg/sabakan-serf-handler
NODE_EXPORTER_VERSION = 0.17.0
NODE_EXPORTER_URL = https://github.com/prometheus/node_exporter/archive/v$(NODE_EXPORTER_VERSION).tar.gz

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

deb: $(DEB)

$(DEB):
	rm -rf $(WORKDIR)
	cp -r debian $(WORKDIR)
	mkdir -p $(WORKDIR)/src $(BINDIR) $(SBINDIR) $(SHAREDIR) $(DOCDIR)/neco $(DOCDIR)/node_exporter
	curl -fsSL $(NODE_EXPORTER_URL) | tar -C $(WORKDIR)/src --strip-components=1 -xzf -
	cd $(WORKDIR)/src; GO111MODULE=on make build
	mv $(WORKDIR)/src/node_exporter $(SBINDIR)/
	cd $(WORKDIR)/src; cp LICENSE NOTICE README.md VERSION $(DOCDIR)/node_exporter/
	rm -rf $(WORKDIR)/src
	sed 's/@VERSION@/$(patsubst v%,%,$(VERSION))/' debian/DEBIAN/control > $(CONTROL)
	GOBIN=$(BINDIR) go install -tags='$(GOTAGS)' $(BIN_PKGS)
	GOBIN=$(SBINDIR) go install -tags='$(GOTAGS)' $(SBIN_PKGS)
	cp etc/* $(SHAREDIR)
	cp -a ignitions $(SHAREDIR)
	cp README.md LICENSE $(DOCDIR)/neco
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
