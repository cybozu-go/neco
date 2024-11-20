
# Makefile for neco

include Makefile.common

COIL_VERSION := $(shell awk '/"coil"/ {match($$6, /[0-9.]+/); print substr($$6,RSTART,RLENGTH)}' artifacts.go)
COIL_OVERLAY = prod
ifeq ($(COIL_PRE), true)
	COIL_CONFIG_SUFFIX = -pre
	COIL_OVERLAY = pre
endif
CILIUM_TAG := $(shell awk '/"cilium"/ {match($$6, /[0-9.]+/); print substr($$6,RSTART,RLENGTH)}' artifacts.go)
CILIUM_OPERATOR_TAG := $(shell awk '/"cilium-operator-generic"/ {match($$6, /[0-9.]+/); print substr($$6,RSTART,RLENGTH)}' artifacts.go)
HUBBLE_RELAY_TAG := $(shell awk '/"hubble-relay"/ {match($$6, /[0-9.]+/); print substr($$6,RSTART,RLENGTH)}' artifacts.go)
CILIUM_CERTGEN_TAG := $(shell awk '/"cilium-certgen"/ {match($$6, /[0-9.]+/); print substr($$6,RSTART,RLENGTH)}' artifacts.go)
CILIUM_OVERLAY = prod
ifeq ($(CILIUM_PRE), true)
	CILIUM_CONFIG_SUFFIX = -pre
	CILIUM_OVERLAY = pre
endif
ifeq ($(CILIUM_DCTEST), true)
	CILIUM_CONFIG_SUFFIX = -dctest
	CILIUM_OVERLAY = dctest
endif
SQUID_VERSION := $(shell awk '/"squid"/ {match($$6, /[0-9.]+/); print substr($$6,RSTART,RLENGTH)}' artifacts.go)
SQUID_EXPORTER_VERSION := $(shell awk '/"squid-exporter"/ {match($$6, /[0-9.]+/); print substr($$6,RSTART,RLENGTH)}' artifacts.go)
SQUID_OVERLAY = prod
ifeq ($(SQUID_PRE), true)
	SQUID_CONFIG_SUFFIX = -pre
	SQUID_OVERLAY = pre
endif
UNBOUND_OVERLAY = prod
ifeq ($(UNBOUND_PRE), true)
	UNBOUND_CONFIG_SUFFIX = -pre
	UNBOUND_OVERLAY = pre
endif
BIN_DIR := $(shell pwd)/bin
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
OPDEB_BINNAMES = argocd hubble jsonnet jsonnetfmt jsonnet-lint kubectl kubeseal kustomize logcli stern tsh kubectl-moco kubectl-accurate amtool yq tempo-cli flamegraph.pl stackcollapse-perf.pl necoperf-cli necoip nsdump clusterdump cmctl vmalert-tool npv
OPDEB_DOCNAMES = argocd hubble jsonnet kubectl kubeseal kustomize logcli stern teleport moco accurate alertmanager yq tempo flamegraph necoperf cmctl vmalert-tool

.PHONY: all
all:
	@echo "Specify one of these targets:"
	@echo
	@echo "    update-coil   - update Coil manifests under etc/."
	@echo "    update-cilium - update Cilium manifests under etc/."
	@echo "    start-etcd    - run etcd on localhost."
	@echo "    stop-etcd     - stop etcd."
	@echo "    test          - run single host tests."
	@echo "    deb           - build Debian package."
	@echo "    tools         - build neco-operation-cli packages."
	@echo "    setup         - install dependencies."

.PHONY: update-coil
update-coil:
	rm -rf /tmp/work-coil
	mkdir -p /tmp/work-coil/cke-patch
	curl -sSfL https://github.com/cybozu-go/coil/archive/v$(COIL_VERSION).tar.gz | tar -C /tmp/work-coil -xzf - --strip-components=1
	cd /tmp/work-coil/v2; sed -i -E 's,^(- config/default),#\1, ; s,^#(- config/cke),\1, ; s,^#(- config/default/pod_security_policy.yaml),\1, ; s,^#(- config/pod/compat_calico.yaml),\1,' kustomization.yaml
	cp etc/netconf.json /tmp/work-coil/v2/netconf.json
	cp -r coil/* /tmp/work-coil/cke-patch
	bin/kustomize build /tmp/work-coil/cke-patch/$(COIL_OVERLAY) > etc/coil$(COIL_CONFIG_SUFFIX).yaml
	rm -rf /tmp/work-coil

.PHONY: update-cilium
update-cilium: helm
	rm -rf /tmp/work-cilium
	mkdir -p /tmp/work-cilium
	git clone --depth 1 --branch v$(shell echo $(CILIUM_TAG) | cut -d \. -f 1,2,3) https://github.com/cilium/cilium /tmp/work-cilium
	cd /tmp/work-cilium
	$(HELM) template /tmp/work-cilium/install/kubernetes/cilium/ \
		--namespace=kube-system \
		--values cilium/$(CILIUM_OVERLAY)/values.yaml \
		--set image.repository=ghcr.io/cybozu/cilium \
		--set image.tag=$(CILIUM_TAG) \
		--set image.useDigest=false \
		--set operator.image.repository=ghcr.io/cybozu/cilium-operator \
		--set operator.image.tag=$(CILIUM_OPERATOR_TAG) \
		--set operator.image.useDigest=false \
		--set hubble.relay.image.repository=ghcr.io/cybozu/hubble-relay \
		--set hubble.relay.image.tag=$(HUBBLE_RELAY_TAG) \
		--set hubble.relay.image.useDigest=false \
		--set certgen.image.repository=ghcr.io/cybozu/cilium-certgen \
		--set certgen.image.tag=$(CILIUM_CERTGEN_TAG) \
		--set certgen.image.useDigest=false > cilium/$(CILIUM_OVERLAY)/upstream.yaml
	bin/kustomize build cilium/$(CILIUM_OVERLAY) > etc/cilium$(CILIUM_CONFIG_SUFFIX).yaml
	rm -rf /tmp/work-cilium

.PHONY: update-squid
update-squid:
	$(call get-unbound-version)
	sed -i -E '/name:.*squid$$/!b;n;s/newTag:.*$$/newTag: $(SQUID_VERSION)/' squid/base/kustomization.yaml
	sed -i -E '/name:.*squid-exporter$$/!b;n;s/newTag:.*$$/newTag: $(SQUID_EXPORTER_VERSION)/' squid/base/kustomization.yaml
	sed -i -E '/name:.*unbound$$/!b;n;s/newTag:.*$$/newTag: $(UNBOUND_VERSION)/' squid/base/kustomization.yaml
	sed -i -E '/name:.*unbound_exporter$$/!b;n;s/newTag:.*$$/newTag: $(UNBOUND_EXPORTER_VERSION)/' squid/base/kustomization.yaml
	bin/kustomize build squid/$(SQUID_OVERLAY) > etc/squid$(SQUID_CONFIG_SUFFIX).yaml

.PHONY: update-unbound
update-unbound:
	$(call get-unbound-version)
	sed -i -E '/name:.*unbound$$/!b;n;s/newTag:.*$$/newTag: $(UNBOUND_VERSION)/' unbound/base/kustomization.yaml
	sed -i -E '/name:.*unbound_exporter$$/!b;n;s/newTag:.*$$/newTag: $(UNBOUND_EXPORTER_VERSION)/' unbound/base/kustomization.yaml
	bin/kustomize build unbound/$(UNBOUND_OVERLAY) > etc/unbound$(UNBOUND_CONFIG_SUFFIX).yaml

HELM := $(shell pwd)/bin/helm
.PHONY: helm
helm: $(HELM) ## Download helm locally if necessary.

$(HELM):
	mkdir -p $(BIN_DIR)
	curl -sSfL https://get.helm.sh/helm-v$(HELM_VERSION)-linux-amd64.tar.gz \
	  | tar xz -C $(BIN_DIR) --strip-components 1 linux-amd64/helm

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
	test -z "$$(custom-checker -restrictpkg.packages=html/template,log -tags='$(GOTAGS)' ./... | tee /dev/stderr)"
	go build -tags='$(GOTAGS)' ./...
	go test -tags='$(GOTAGS)' -race -v ./...
	RUN_COMPACTION_TEST=yes go test -tags='$(GOTAGS)' -race -v -run=TestEtcdCompaction ./worker/
	go vet -tags='$(GOTAGS)' ./...

.PHONY: check-generate
check-generate:
	$(MAKE) -C ignition-template all
	$(MAKE) update-coil
	$(MAKE) update-coil COIL_PRE=true
	$(MAKE) update-cilium
	$(MAKE) update-cilium CILIUM_PRE=true
	$(MAKE) update-cilium CILIUM_DCTEST=true
	$(MAKE) update-squid
	$(MAKE) update-squid SQUID_PRE=true
	$(MAKE) update-unbound
	$(MAKE) update-unbound UNBOUND_PRE=true
	go mod tidy
	git diff --exit-code --name-only

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
	GOBIN=$(BINDIR) CGO_ENABLED=0 go install -ldflags="-s -w" -tags='$(GOTAGS)' $(BIN_PKGS)
	CGO_ENABLED=0 go build -o $(BINDIR)/sabakan-state-setter -ldflags="-s -w" -tags='$(GOTAGS)' ./pkg/sabakan-state-setter/cmd
	CGO_ENABLED=0 go build -o $(BINDIR)/neco-rebooter -ldflags="-s -w" -tags='$(GOTAGS)' ./pkg/neco-rebooter/cmd
	GOBIN=$(SBINDIR) CGO_ENABLED=0 go install -ldflags="-s -w" -tags='$(GOTAGS)' $(SBIN_PKGS)
	tar -c -f $(LIBEXECDIR)/neco-operation-cli.tgz -z -C $(BINDIR) $(OPDEB_BINNAMES)
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

.PHONY: setup
setup:
	$(SUDO) apt-get update
	$(SUDO) apt-get -y install --no-install-recommends $(PACKAGES)
	mkdir -p bin
	bin/curl-github -sSfL https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2Fv$(KUSTOMIZE_VERSION)/kustomize_v$(KUSTOMIZE_VERSION)_linux_amd64.tar.gz | tar -xz -C bin
	chmod a+x bin/kustomize

.PHONY: clean
clean:
	$(MAKE) -f Makefile.tools clean
	rm -rf $(ETCD_DIR) $(WORKDIR) $(DEB) $(OPWORKDIR) $(OPWORKWINDIR) $(OPWORKMACDIR) $(OP_DEB) $(OP_WIN_ZIP) $(OP_MAC_ZIP)

define get-unbound-version
$(eval UNBOUND_VERSION := $(shell curl -sSfL https://raw.githubusercontent.com/cybozu-go/cke/v$(CKE_VERSION)/images.go | awk -F'"' '/UnboundImage/ {match($$2, /([0-9]+).([0-9]+).([0-9]+).([0-9]+)/); print substr($$2,RSTART,RLENGTH)}'))
$(eval UNBOUND_EXPORTER_VERSION := $(shell curl -sSfL https://raw.githubusercontent.com/cybozu-go/cke/v$(CKE_VERSION)/images.go | awk -F'"' '/UnboundExporterImage/ {match($$2, /([0-9]+).([0-9]+).([0-9]+).([0-9]+)/); print substr($$2,RSTART,RLENGTH)}'))
endef
