# Makefile for neco-test

# configuration variables
BOOT0 = 10.72.48.0
BOOT1 = 10.72.48.1
BOOT2 = 10.72.48.2
BOOT3 = 10.72.48.3
K8S_CTRLPLANE0 = 10.69.0.4
K8S_CTRLPLANE1 = 10.69.0.196
K8S_CTRLPLANE2 = 10.69.1.132
PLACEMAT = /usr/bin/placemat
GINKGO = env GO111MODULE=on GOFLAGS=-mod=vendor $(GOPATH)/bin/ginkgo --failFast -v
SUITE = ./functions
PLACEMAT_DATADIR = /var/scratch/placemat
VM_SHARED_DIR = /mnt/placemat
MENU = menu.yml

export BOOT0 BOOT1 BOOT2 BOOT3 PLACEMAT GINKGO SUITE

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
AUTOCONFIG = $(SABAKAN_DIR)/auto-config
BMC_USER = $(SABAKAN_DIR)/bmc-user.json
OLD_DEB = $(SABAKAN_DIR)/neco_$(OLDVER)_amd64.deb
NEW_DEB = $(SABAKAN_DIR)/neco_$(NEWVER)_amd64.deb
SABAKAN_CONTENTS = $(AUTOCONFIG) $(OLD_DEB) $(NEW_DEB) $(BMC_USER)
SSH_PRIVKEY = $(realpath ./dctest_key)
BOB_PRIVKEY = $(realpath ./bob_key)
BOB_PUBKEY = $(realpath ./bob_key.pub)

export SSH_PRIVKEY BOB_PRIVKEY BOB_PUBKEY

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
	$(GOPATH)/bin/placemat-menu -f $(MENU) -o $(OUTPUT)

$(SABAKAN_DIR)/auto-config: auto-config $(DHCP_JSON)
	cp $< $@

$(SABAKAN_DIR)/neco_$(OLDVER)_amd64.deb: $(OPERATION_SSH_DIR)
	cd ..; make deb VERSION=$(OLDVER) DEST=$(realpath $(SABAKAN_DIR))

$(SABAKAN_DIR)/neco_$(NEWVER)_amd64.deb: $(OPERATION_SSH_DIR)
	cd ..; make deb VERSION=$(NEWVER) DEST=$(realpath $(SABAKAN_DIR)) TAGS=new

$(SABAKAN_DIR)/bmc-user.json: bmc-user.json $(DHCP_JSON)
	cp $< $@

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
	sudo systemd-run --unit=placemat.service $(PLACEMAT) -enable-virtfs $(realpath output/cluster.yml)
	@echo 'Placemat is running.  To kill it, do "make stop".'

stop:
	sudo systemctl stop placemat.service
	sudo systemctl disable placemat.service
	sudo rm /run/systemd/transient/placemat.service
	sudo systemctl daemon-reload
	sudo systemctl reset-failed placemat.service

test-light:
	./test-light.sh "$(TARGET)"

test-release: $(OUTPUT)/$(CUSTOM_UBUNTU) $(AUTOCONFIG) $(OPERATION_SSH_DIR)
	cd ..; make deb VERSION=$(OLDVER) DEST=$(realpath $(SABAKAN_DIR)) TAGS=release
	sudo rm -rf $(PLACEMAT_DATADIR)
	sudo rm -rf $(VM_SHARED_DIR)
	SUITE=bootstrap ./test.sh

clean:
	rm -rf $(OUTPUT)

fullclean: clean
	rm -rf $(CUSTOM_UBUNTU)

setup:
	cd ../ && $(MAKE) setup
	GO111MODULE=off	go get -d -u github.com/cybozu-go/placemat-menu
	GO111MODULE=off go get github.com/onsi/ginkgo/ginkgo
	cd $(GOPATH)/src/github.com/cybozu-go/placemat-menu; GO111MODULE=on make

.PHONY:	all test placemat test-light test-release clean fullclean setup
