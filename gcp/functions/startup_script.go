package functions

import (
	"errors"
	"fmt"
)

const accountJSONName = "neco-apps-gcp-account"

// NecoStartupScriptBuilder creates startup-script builder to run dctest
type NecoStartupScriptBuilder struct {
	withFluentd    bool
	necoBranch     string
	necoAppsBranch string
}

// NewNecoStartupScriptBuilder creates NecoStartupScriptBuilder
func NewNecoStartupScriptBuilder() *NecoStartupScriptBuilder {
	return &NecoStartupScriptBuilder{}
}

// WithFluentd enables fluentd
func (b *NecoStartupScriptBuilder) WithFluentd() *NecoStartupScriptBuilder {
	b.withFluentd = true
	return b
}

// WithNeco sets which branch to run neco
func (b *NecoStartupScriptBuilder) WithNeco(branch string) *NecoStartupScriptBuilder {
	b.necoBranch = branch
	return b
}

// WithNecoApps sets which branch to run neco-apps
func (b *NecoStartupScriptBuilder) WithNecoApps(branch string) (*NecoStartupScriptBuilder, error) {
	if len(b.necoBranch) == 0 {
		return nil, errors.New("please specify neco branch to run neco-apps")
	}
	b.necoAppsBranch = branch
	return b, nil
}

// Build  builds startup script
func (b *NecoStartupScriptBuilder) Build() string {
	s := `#! /bin/sh
# mkfs and mount local SSD on /var/scratch
mkfs -t ext4 -F /dev/disk/by-id/google-local-ssd-0
mkdir -p /var/scratch
mount -t ext4 /dev/disk/by-id/google-local-ssd-0 /var/scratch
chmod 1777 /var/scratch
`

	if b.withFluentd {
		s += `
# Run fluentd to export syslog to Cloud Logging
curl -sSO https://dl.google.com/cloudagents/add-logging-agent-repo.sh
bash add-logging-agent-repo.sh
apt-get update
apt-cache madison google-fluentd
apt-get install -y google-fluentd
apt-get install -y google-fluentd-catch-all-config-structured
service google-fluentd start
# This line is needed to ensure that fluentd is running
service google-fluentd restart
`
	}

	if len(b.necoBranch) > 0 {
		s += fmt.Sprintf(`
# Set environment variables
HOME=/root
GOPATH=${HOME}/go
GO111MODULE=on
PATH=${PATH}:/usr/local/go/bin:${GOPATH}/bin
export HOME GOPATH GO111MODULE PAT

# Run neco
mkdir -p ${GOPATH}/src/github.com/cybozu-go
cd ${GOPATH}/src/github.com/cybozu-go
git clone https://github.com/cybozu-go/neco
cd ${GOPATH}/src/github.com/cybozu-go/neco/dctest
git checkout %s
make setup placemat test SUITE=./bootstrap
`, b.necoBranch)
	}

	if len(b.necoAppsBranch) > 0 {
		s += fmt.Sprintf(`
# Run neco-apps
cd ${GOPATH}/src/github.com/cybozu-go
git clone https://github.com/cybozu-go/neco-apps
cd ${GOPATH}/src/github.com/cybozu-go/neco-apps/test
git checkout %s
gcloud secrets versions access latest --secret="%s" > account.json
make setup dctest BOOTSTRAP=1
`, b.necoAppsBranch, accountJSONName)
	}
	return s
}
