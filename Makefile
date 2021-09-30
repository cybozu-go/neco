
# Makefile for neco

include Makefile.common

COIL_VERSION := $(shell awk '/"coil"/ {match($$6, /[0-9.]+/); print substr($$6,RSTART,RLENGTH)}' artifacts.go)
LSB_DISTRIB_RELEASE := $(shell . /etc/lsb-release ; echo $$DISTRIB_RELEASE)

FAKEROOT = fakeroot
ETCD_DIR = /tmp/neco-etcd
TAGS =

### for Go
GOTAGS = $(TAGS)

### for debian package
PACKAGES = fakeroot pkg-config libpam0g-dev unzip zip wget
VERSION = 0.0.1-main
DEST = .
DEB = neco_$(VERSION)_amd64.deb
OP_DEB = neco-operation-cli-linux_$(VERSION)_amd64.deb
OP_WIN_ZIP = neco-operation-cli-windows_$(VERSION)_amd64.zip
OP_MAC_ZIP = neco-operation-cli-mac_$(VERSION)_amd64.zip
DEBBUILD_FLAGS = -Znone
BIN_PKGS = ./pkg/neco
SBIN_PKGS = ./pkg/neco-updater ./pkg/neco-worker
OPDEB_BINNAMES = argocd kubectl kubeseal logcli stern tsh kubectl-moco kubectl-accurate amtool
OPDEB_DOCNAMES = argocd kubectl kubeseal logcli stern teleport moco alertmanager

.PHONY: all
all:
	@echo "Specify one of these targets:"
	@echo
	@echo "    update-coil - update Coil manifests under etc/."
	@echo "    start-etcd  - run etcd on localhost."
	@echo "    stop-etcd   - stop etcd."
	@echo "    test        - run single host tests."
	@echo "    deb         - build Debian package."
	@echo "    tools       - build neco-operation-cli packages."
	@echo "    setup       - install dependencies."

.PHONY: update-coil
update-coil:
	rm -rf /tmp/work-coil
	mkdir -p /tmp/work-coil/cke-patch
	curl -sfL https://github.com/cybozu-go/coil/archive/v$(COIL_VERSION).tar.gz | tar -C /tmp/work-coil -xzf - --strip-components=1
	cd /tmp/work-coil/v2; sed -i -E 's,^(- config/default),#\1, ; s,^#(- config/cke),\1, ; s,^#(- config/default/pod_security_policy.yaml),\1, ; s,^#(- config/pod/compat_calico.yaml),\1,' kustomization.yaml
	cp etc/netconf.json /tmp/work-coil/v2/netconf.json
	cp coil/* /tmp/work-coil/cke-patch
	bin/kustomize build /tmp/work-coil/cke-patch > etc/coil.yaml
	rm -rf /tmp/work-coil

.PHONY: start-etcd
start-etcd:
	systemd-run --user --unit neco-etcd.service etcd --data-dir $(ETCD_DIR)

.PHONY: stop-etcd
stop-etcd:
	systemctl --user stop neco-etcd.service

.PHONY: test
test:
	test -z "$$(gofmt -s -l . | grep -v '^build/' | tee /dev/stderr)"
	staticcheck -tags='$(GOTAGS)' ./...
	nilerr -tags='$(GOTAGS)' ./...
	test -z "$$(custom-checker -restrictpkg.packages=html/template,log -tags='$(GOTAGS)' ./... | tee /dev/stderr)"
	go build -tags='$(GOTAGS)' ./...
	go test -tags='$(GOTAGS)' -race -v ./...
	RUN_COMPACTION_TEST=yes go test -tags='$(GOTAGS)' -race -v -run=TestEtcdCompaction ./worker/
	go vet -tags='$(GOTAGS)' ./...

.PHONY: check-coil-version
check-coil-version:
	grep -F $(COIL_VERSION) etc/coil.yaml

.PHONY: deb
deb: $(DEB)

.PHONY: tools
tools: $(OP_DEB) $(OP_WIN_ZIP) $(OP_MAC_ZIP)

.PHONY: setup-tools
setup-tools:
	$(MAKE) -f Makefile.tools

.PHONY: setup-files-for-deb
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
	for BINNAME in $(OPDEB_BINNAMES); do \
		cp $(BINDIR)/$$BINNAME $(OPBINDIR) || exit 1 ; \
	done
	for DOCNAME in $(OPDEB_DOCNAMES); do \
		cp -r $(DOCDIR)/$$DOCNAME $(OPDOCDIR) || exit 1 ; \
	done
	echo $(VERSION) > $(OPWORKDIR)/VERSION
	$(FAKEROOT) dpkg-deb --build $(DEBBUILD_FLAGS) $(OPWORKDIR) $(DEST)

$(OP_WIN_ZIP): setup-tools
	mkdir -p $(OPWORKWINDIR)
	echo $(VERSION) > $(OPWORKWINDIR)/VERSION
	cd $(OPWORKWINDIR) && zip -r $(abspath .)/$@ *

$(OP_MAC_ZIP): setup-tools
	mkdir -p $(OPWORKMACDIR)
	echo $(VERSION) > $(OPWORKMACDIR)/VERSION
	cd $(OPWORKMACDIR) && zip -r $(abspath .)/$@ *

.PHONY: gcp-deb
gcp-deb: setup-files-for-deb
	cp dctest/passwd.yml $(SHAREDIR)/ignitions/common/passwd.yml
	sed -i -e "s/TimeoutStartSec=infinity/TimeoutStartSec=1200/g" $(SHAREDIR)/ignitions/common/systemd/setup-var.service
	$(FAKEROOT) dpkg-deb --build $(DEBBUILD_FLAGS) $(WORKDIR) $(DEST)

.PHONY: git-neco
git-neco:
	go install ./pkg/git-neco

.PHONY: setup
setup:
	$(SUDO) apt-get update
	$(SUDO) apt-get -y install --no-install-recommends $(PACKAGES)
	mkdir -p bin
	curl -sfL https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2Fv$(KUSTOMIZE_VERSION)/kustomize_v$(KUSTOMIZE_VERSION)_linux_amd64.tar.gz | tar -xz -C bin
	chmod a+x bin/kustomize

.PHONY: clean
clean:
	$(MAKE) -f Makefile.tools clean
	rm -rf $(ETCD_DIR) $(WORKDIR) $(DEB) $(OPWORKDIR) $(OPWORKWINDIR) $(OPWORKMACDIR) $(OP_DEB) $(OP_WIN_ZIP) $(OP_MAC_ZIP)
