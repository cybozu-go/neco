#!/bin/sh -ex

. $(dirname $0)/env
SUITE_NAME=$1
TAG_NAME=$2
DATACENTER=$3

# Create GCE instance
$GCLOUD compute instances delete ${INSTANCE_NAME} --zone ${ZONE} --quiet || true
$GCLOUD compute instances create ${INSTANCE_NAME} \
  --zone ${ZONE} \
  --machine-type ${MACHINE_TYPE} \
  --image vmx-enabled \
  --boot-disk-type ${DISK_TYPE} \
  --boot-disk-size ${BOOT_DISK_SIZE} \
  --local-ssd interface=nvme \
  --local-ssd interface=nvme \
  --local-ssd interface=nvme \
  --local-ssd interface=nvme

# Run data center test
for i in $(seq 300); do
  if $GCLOUD compute ssh --zone=${ZONE} cybozu@${INSTANCE_NAME} --command=date 2>/dev/null; then
    break
  fi
  sleep 1
done

cat >run.sh <<EOF
#!/bin/sh -ex

# mkfs and mount local SSD on /var/scratch
mkfs -t ext4 -F /dev/disk/by-id/google-local-nvme-ssd-0
mkdir -p /var/scratch
mount -t ext4 /dev/disk/by-id/google-local-nvme-ssd-0 /var/scratch
chmod 1777 /var/scratch

# Set environment variables
GO111MODULE=on
export GO111MODULE
GOPATH=\${HOME}/go
export GOPATH
PATH=/usr/local/go/bin:\${GOPATH}/bin:\${PATH}
export PATH

NECO_DIR=\${GOPATH}/src/github.com/cybozu-go/neco
TEMP_DIR=/tmp/release

# Prepare neco repository
mkdir -p \${NECO_DIR}
cd \${NECO_DIR}
tar xzf /home/cybozu/neco.tgz

# Prepare files
cp \${NECO_DIR}/github-token \${NECO_DIR}/dctest/
cp \${NECO_DIR}/secrets \${NECO_DIR}/dctest/
cp /assets/cybozu-ubuntu-18.04-server-cloudimg-amd64.img \${NECO_DIR}/dctest/

# Check out released sources which are used in the specified datacenter
if [ -n "${DATACENTER}" ]; then
  go install -mod=vendor ./pkg/find-installed-release
  RELEASE=\$(find-installed-release ${DATACENTER})
  git worktree add \${TEMP_DIR} release-\$RELEASE
  cd \${TEMP_DIR}

  # Prepare files
  cp \${NECO_DIR}/github-token \${TEMP_DIR}/dctest/
  cp \${NECO_DIR}/secrets \${TEMP_DIR}/dctest/
  cp /assets/cybozu-ubuntu-18.04-server-cloudimg-amd64.img \${TEMP_DIR}/dctest/
fi

# Run dctest
cd dctest
make setup
make placemat TAGS=${TAG_NAME}
sleep 3
exec make test TAGS=${TAG_NAME} SUITE=${SUITE_NAME} DATACENTER=${DATACENTER}
EOF
chmod +x run.sh

tar czf /tmp/neco.tgz .
$GCLOUD compute scp --zone=${ZONE} /tmp/neco.tgz cybozu@${INSTANCE_NAME}:
$GCLOUD compute scp --zone=${ZONE} run.sh cybozu@${INSTANCE_NAME}:
set +e
$GCLOUD compute ssh --zone=${ZONE} cybozu@${INSTANCE_NAME} --command='sudo -H /home/cybozu/run.sh'
exit $?
