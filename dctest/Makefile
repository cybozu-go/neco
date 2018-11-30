# Makefile for neco-test

# configuration variables
BOOT0 = 10.72.48.0
BOOT1 = 10.72.48.1
BOOT2 = 10.72.48.2
BOOT3 = 10.72.48.3
K8S_CTRLPLANE0 = 10.69.0.4
K8S_CTRLPLANE1 = 10.69.0.196
K8S_CTRLPLANE2 = 10.69.1.132

PLACEMAT = $(GOPATH)/bin/placemat
GINKGO = env GO111MODULE=on GOFLAGS=-mod=vendor $(GOPATH)/bin/ginkgo --failFast -v
PLACEMAT_DATADIR = /var/scratch/placemat
VM_SHARED_DIR = /mnt/placemat

export BOOT0 BOOT1 BOOT2 BOOT3 PLACEMAT GINKGO

CUSTOM_UBUNTU = cybozu-ubuntu-18.04-server-cloudimg-amd64.img
CUSTOM_UBUNTU_API_URL = https://api.github.com/repos/cybozu/neco-ubuntu/releases/latest
CUSTOM_UBUNTU_DISK_SIZE = 30G

# non-configuration variables
OUTPUT = ./output
SABAKAN_DIR = $(OUTPUT)/sabakan
OPERATION_SSH_DIR = $(OUTPUT)/operation-ssh
DHCP_JSON = $(SABAKAN_DIR)/dhcp.json
OLDVER = 0.0.1
NEWVER = 0.0.2
SABAKAN_CONTENTS = $(SABAKAN_DIR)/auto-config \
	$(SABAKAN_DIR)/neco_$(OLDVER)_amd64.deb \
	$(SABAKAN_DIR)/neco_$(NEWVER)_amd64.deb \

SSH_PRIVKEY = ./dctest_key

export SSH_PRIVKEY

all: test lint

# CUSTOM_UBUNTU
$(CUSTOM_UBUNTU):
	curl -fSL -o $@ $$(curl -fsSL $(CUSTOM_UBUNTU_API_URL) | jq -r '.assets[] | select (.name=="$(CUSTOM_UBUNTU)") | .browser_download_url')

$(OUTPUT)/$(CUSTOM_UBUNTU): $(CUSTOM_UBUNTU)
	qemu-img resize $< $(CUSTOM_UBUNTU_DISK_SIZE)  # Do this here because CUSTOM_IMAGE is included in vmx-enabled image.
	mkdir -p $(OUTPUT)
	rm -f $@
	ln $< $@
	touch $@

# SABAKAN_CONTENTS
$(DHCP_JSON):
	$(GOPATH)/bin/placemat-menu -f menu.yml -o $(OUTPUT)

$(SABAKAN_DIR)/auto-config: auto-config $(DHCP_JSON)
	cp $< $@

$(SABAKAN_DIR)/neco_$(OLDVER)_amd64.deb: $(OPERATION_SSH_DIR)
	cd ..; make deb VERSION=$(OLDVER) DEST=$(realpath $(SABAKAN_DIR))

$(SABAKAN_DIR)/neco_$(NEWVER)_amd64.deb: $(OPERATION_SSH_DIR)
	cd ..; make deb VERSION=$(NEWVER) DEST=$(realpath $(SABAKAN_DIR)) TAGS=new

# workaround for placemat-menu
$(OPERATION_SSH_DIR):
	mkdir -p $(OPERATION_SSH_DIR)
	cp dctest_key $(OPERATION_SSH_DIR)/boot_key

# Targets
test: $(OUTPUT)/$(CUSTOM_UBUNTU) $(SABAKAN_CONTENTS)
	sudo rm -rf $(PLACEMAT_DATADIR)
	sudo rm -rf $(VM_SHARED_DIR)
	./test.sh

placemat: $(OUTPUT)/$(CUSTOM_UBUNTU) $(SABAKAN_CONTENTS)
	sudo rm -rf $(PLACEMAT_DATADIR)
	sudo rm -rf $(VM_SHARED_DIR)
	sudo -b $(PLACEMAT) -enable-virtfs output/cluster.yml
	@echo 'Placemat is running.  To kill it, do "sudo pkill placemat".'

test-light:
	placemat_pid=$$(echo $$(pgrep placemat) | tr " " ",") ; \
	operation_pid=$$(pgrep -P $$placemat_pid -f operation) ; \
	sudo -E nsenter -t $$operation_pid -n sh -c "export PATH=$$PATH; $(GINKGO) -focus=\"${TARGET}\""

clean:
	rm -rf $(OUTPUT)

fullclean: clean
	rm -rf $(CUSTOM_UBUNTU)

setup:
	GO111MODULE=off go get -t -d ./... \
		github.com/cybozu-go/placemat-menu
	GO111MODULE=off go get github.com/onsi/ginkgo/ginkgo \
		github.com/cybozu-go/placemat/...
	cd $(GOPATH)/src/github.com/cybozu-go/placemat-menu; GO111MODULE=on make

.PHONY:	all test placemat test-light clean fullclean setup
