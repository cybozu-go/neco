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
PACKAGES := fakeroot btrfs-tools pkg-config libdevmapper-dev
VERSION = 0.0.1-master
DEST = .
DEB = neco_$(VERSION)_amd64.deb
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
	ineffassign .
	test -z "$$(nilerr ./... 2>&1 | tee /dev/stderr)"
	test -z "$$(restrictpkg -packages=html/template,log ./... 2>&1 | tee /dev/stderr)"
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

setup-files-for-deb:
	$(MAKE) -f Makefile.tools SUDO=$(SUDO)
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
	GO111MODULE=off go get -u golang.org/x/lint/golint \
		github.com/rakyll/statik \
		github.com/cybozu/neco-containers/golang/restrictpkg/cmd/restrictpkg
	$(SUDO) apt-get update
	$(SUDO) apt-get -y install --no-install-recommends $(PACKAGES)

clean:
	$(MAKE) -f Makefile.tools clean
	rm -rf $(ETCD_DIR) $(WORKDIR) $(DEB)

.PHONY:	all start-etcd stop-etcd test mod deb setup-files-for-deb gcp-deb necogcp git-neco setup clean
