# Makefile for neco

include Makefile.common

FAKEROOT = fakeroot
ETCD_DIR = /tmp/neco-etcd
TAGS =

### for Go
GOFLAGS = -mod=vendor
GOTAGS = $(TAGS) containers_image_openpgp containers_image_ostree_stub
export GOFLAGS

### for debian package
PACKAGES := fakeroot btrfs-tools pkg-config libdevmapper-dev libostree-dev libgpgme-dev libgpgme11 unzip zip
VERSION = 0.0.1-master
DEST = .
DEB = neco_$(VERSION)_amd64.deb
OP_DEB = neco-operation-cli-linux_$(VERSION)_amd64.deb
OP_ZIP = neco-operation-cli-windows_$(VERSION)_amd64.zip
DEBBUILD_FLAGS = -Znone
BIN_PKGS = ./pkg/neco
SBIN_PKGS = ./pkg/neco-updater ./pkg/neco-worker

STATIK = gcp/statik/statik.go
STATIK_FILES := $(shell find gcp/public -type f)

all:
	@echo "Specify one of these targets:"
	@echo
	@echo "    start-etcd  - run etcd on localhost."
	@echo "    stop-etcd   - stop etcd."
	@echo "    test        - run single host tests."
	@echo "    mod         - update and vendor Go modules."
	@echo "    deb         - build Debian package."
	@echo "    tools       - build neco-operation-cli packages."
	@echo "    setup       - install dependencies."

start-etcd:
	systemd-run --user --unit neco-etcd.service etcd --data-dir $(ETCD_DIR)

stop-etcd:
	systemctl --user stop neco-etcd.service

$(STATIK): $(STATIK_FILES)
	mkdir -p $(dir $(STATIK))
	go generate ./pkg/necogcp/...

test: $(STATIK)
	test -z "$$(gofmt -s -l . | grep -v '^vendor/\|^menu/assets.go\|^build/' | tee /dev/stderr)"
	test -z "$$(golint $$(go list -tags='$(GOTAGS)' ./... | grep -v /vendor/) | grep -v '/dctest/.*: should not use dot imports' | tee /dev/stderr)"
	test -z "$$(nilerr $$(go list -tags='$(GOTAGS)' ./... | grep -v /vendor/) 2>&1 | tee /dev/stderr)"
	test -z "$$(custom-checker -restrictpkg.packages=html/template,log $$(go list -tags='$(GOTAGS)' ./... | grep -v /vendor/ ) 2>&1 | tee /dev/stderr)"
	ineffassign .
	go build -tags='$(GOTAGS)' ./...
	go test -tags='$(GOTAGS)' -race -v ./...
	RUN_COMPACTION_TEST=yes go test -tags='$(GOTAGS)' -race -v -run=TestEtcdCompaction ./worker/
	go vet -tags='$(GOTAGS)' ./...

mod:
	go mod tidy
	go mod vendor
	git add -f vendor
	git add go.mod

deb: $(DEB)

tools: $(OP_DEB) $(OP_ZIP)

setup-tools:
	$(MAKE) -f Makefile.tools SUDO=$(SUDO)

setup-files-for-deb: setup-tools
	cp -r debian/* $(WORKDIR)
	mkdir -p $(WORKDIR)/src $(BINDIR) $(SBINDIR) $(SHAREDIR) $(DOCDIR)/neco
	sed 's/@VERSION@/$(patsubst v%,%,$(VERSION))/' debian/DEBIAN/control > $(CONTROL)
	GOBIN=$(BINDIR) go install -tags='$(GOTAGS)' $(BIN_PKGS)
	go build -o $(BINDIR)/sabakan-state-setter -tags='$(GOTAGS)' ./pkg/sabakan-state-setter/cmd
	GOBIN=$(SBINDIR) go install -tags='$(GOTAGS)' $(SBIN_PKGS)
	cp etc/* $(SHAREDIR)
	cp -a ignitions $(SHAREDIR)
	cp README.md LICENSE $(DOCDIR)/neco
	chmod -R g-w $(WORKDIR)

$(DEB): setup-files-for-deb
	$(FAKEROOT) dpkg-deb --build $(DEBBUILD_FLAGS) $(WORKDIR) $(DEST)

$(OP_DEB): setup-files-for-deb
	mkdir -p $(OPBINDIR) $(OPDOCDIR) $(OPWORKDIR)/DEBIAN
	sed 's/@VERSION@/$(patsubst v%,%,$(VERSION))/; /Package: neco/s/$$/-operation-cli-linux/; s/Continuous delivery tool/Operation tools/' debian/DEBIAN/control > $(OPCONTROL)
	for BINNAME in argocd kubectl kustomize stern teleport; do \
		cp $(BINDIR)/$$BINNAME $(OPBINDIR) ; \
		cp -r $(DOCDIR)/$$BINNAME $(OPDOCDIR) ; \
	done
	$(MAKE) -f Makefile.tools SUDO=$(SUDO) BINDIR=$(OPBINDIR) DOCDIR=$(OPDOCDIR) teleport
	$(FAKEROOT) dpkg-deb --build $(DEBBUILD_FLAGS) $(OPWORKDIR) $(DEST)

$(OP_ZIP): setup-tools
	cd $(WORKDIR)/windows/ && zip -r $(abspath .)/$@ *

gcp-deb: setup-files-for-deb
	cp dctest/passwd.yml $(SHAREDIR)/ignitions/common/passwd.yml
	sed -i -e "s/TimeoutStartSec=infinity/TimeoutStartSec=1200/g" $(SHAREDIR)/ignitions/common/systemd/setup-var.service
	$(FAKEROOT) dpkg-deb --build $(DEBBUILD_FLAGS) $(WORKDIR) $(DEST)

necogcp: $(STATIK)
	go install ./pkg/necogcp

git-neco:
	go install ./pkg/git-neco

setup:
	go install github.com/rakyll/statik
	$(SUDO) apt-get update
	$(SUDO) apt-get -y install --no-install-recommends $(PACKAGES)

clean:
	$(MAKE) -f Makefile.tools clean
	rm -rf $(ETCD_DIR) $(WORKDIR) $(DEB) $(OPWORKDIR) $(OP_DEB) $(OP_ZIP)

.PHONY:	all start-etcd stop-etcd test mod deb setup-tools setup-files-for-deb gcp-deb necogcp git-neco setup clean tools
