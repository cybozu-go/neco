#!/bin/sh -ex

SUITE=$1
DATACENTER=$2

. $(dirname $0)/env

delete_instance() {
  if [ $RET -ne 0 ]; then
    # do not delete GCP instance upon test failure to help debugging.
    return
  fi
  if [ -n "${DATACENTER}" ]; then
    # do not delete GCP instance because it is used by subsequent upgrading test.
    return
  fi
  $GCLOUD compute instances delete ${INSTANCE_NAME} --zone ${ZONE} || true
}

# Create GCE instance
$GCLOUD compute instances delete ${INSTANCE_NAME} --zone ${ZONE} --quiet || true
$GCLOUD compute instances create ${INSTANCE_NAME} \
  --zone ${ZONE} \
  --machine-type ${MACHINE_TYPE} \
  --image ${IMAGE_NAME} \
  --boot-disk-type ${DISK_TYPE} \
  --boot-disk-size ${BOOT_DISK_SIZE} \
  --local-ssd interface=scsi

RET=0
trap delete_instance INT QUIT TERM 0

# Run data center test
for i in $(seq 300); do
  if $GCLOUD compute ssh --zone=${ZONE} cybozu@${INSTANCE_NAME} --command=date 2>/dev/null; then
    break
  fi
  sleep 1
done

cat >run.sh <<EOF
#!/bin/sh -e

# mkfs and mount local SSD on /var/scratch
mkfs -t ext4 -F /dev/disk/by-id/google-local-ssd-0
mkdir -p /var/scratch
mount -t ext4 /dev/disk/by-id/google-local-ssd-0 /var/scratch
chmod 1777 /var/scratch

GOPATH=\$HOME/go
export GOPATH
PATH=/usr/local/go/bin:\$GOPATH/bin:\$PATH
export PATH

# Prepare neco repository
mkdir -p \$GOPATH/src/github.com/cybozu-go/neco
cd \$GOPATH/src/github.com/cybozu-go/neco
tar xzf /home/cybozu/neco.tgz
cp secrets dctest/
if [ -n "${DATACENTER}" ]; then
  env GO111MODULE=on go install -mod=vendor ./pkg/find-installed-release
  RELEASE=\$(find-installed-release ${DATACENTER})
  git worktree add /tmp/release release-\$RELEASE
  cp secrets /tmp/release/dctest/
  OLD_WD=\$(pwd)
  cd /tmp/release
fi

# Run dctest
cd dctest
cp /assets/cybozu-ubuntu-18.04-server-cloudimg-amd64.img .
export GO111MODULE=on
make setup
make placemat MENU=highcpu-menu.yml TAGS=release
sleep 3
make test TAGS=release SUITE=${SUITE} DATACENTER=${DATACENTER}
if [ -n "${DATACENTER}" ]; then
  cd \${OLD_WD}/dctest
  cp /assets/cybozu-ubuntu-18.04-server-cloudimg-amd64.img .
fi
EOF
chmod +x run.sh

tar czf /tmp/neco.tgz .
$GCLOUD compute scp --zone=${ZONE} /tmp/neco.tgz cybozu@${INSTANCE_NAME}:
$GCLOUD compute scp --zone=${ZONE} run.sh cybozu@${INSTANCE_NAME}:
set +e
$GCLOUD compute ssh --zone=${ZONE} cybozu@${INSTANCE_NAME} --command='sudo /home/cybozu/run.sh'
RET=$?

exit $RET
