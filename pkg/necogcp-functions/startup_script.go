package necogcpfunctions

// MakeStartupScript creates startup-script to run dctest
func MakeStartupScript(
	accountJSONPath string,
	necoBranch string,
	necoAppsBranch string,
) string {
	script := `#! /bin/sh

# Run fluentd to export syslog to Cloud Logging
curl -sSO https://dl.google.com/cloudagents/add-logging-agent-repo.sh
bash add-logging-agent-repo.sh
apt-get update
apt-cache madison google-fluentd
apt-get install -y google-fluentd
apt-get install -y google-fluentd-catch-all-config-structured
service google-fluentd start
service google-fluentd restart

# Set environment variables
HOME=/root
GOPATH=${HOME}/go
GO111MODULE=on
PATH=${PATH}:/usr/local/go/bin:${GOPATH}/bin
export HOME GOPATH GO111MODULE PATH

# mkfs and mount local SSD on /var/scratch
mkfs -t ext4 -F /dev/disk/by-id/google-local-ssd-0
mkdir -p /var/scratch
mount -t ext4 /dev/disk/by-id/google-local-ssd-0 /var/scratch
chmod 1777 /var/scratch

# Run test
mkdir -p ${GOPATH}/src/github.com/cybozu-go
cd ${GOPATH}/src/github.com/cybozu-go
git clone https://github.com/cybozu-go/neco
cd neco/dctest
make setup placemat test SUITE=./bootstrap
`
	return script
}
