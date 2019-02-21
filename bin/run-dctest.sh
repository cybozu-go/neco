#!/bin/sh -ex

SUITE=$1
DATACENTER=$2

. $(dirname $0)/env

delete_instance() {
  if [ $RET -ne 0 ]; then
    # do not delete GCP instance upon test failure to help debugging.
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

# Run dctest
GOPATH=\$HOME/go
export GOPATH
PATH=/usr/local/go/bin:\$GOPATH/bin:\$PATH
export PATH

git clone https://github.com/${CIRCLE_PROJECT_USERNAME}/${CIRCLE_PROJECT_REPONAME} \$HOME/go/src/github.com/${CIRCLE_PROJECT_USERNAME}/${CIRCLE_PROJECT_REPONAME}
cd \$HOME/go/src/github.com/${CIRCLE_PROJECT_USERNAME}/${CIRCLE_PROJECT_REPONAME}
git checkout -qf ${CIRCLE_SHA1}

cd dctest
cp /home/cybozu/secrets .
cp /home/cybozu/github-token .
cp /assets/cybozu-ubuntu-18.04-server-cloudimg-amd64.img .
export GO111MODULE=on
make setup
make placemat
sleep 3
exec make test MENU=highcpu-menu.yml SUITE=${SUITE} DATACENTER=${DATACENTER}
EOF
chmod +x run.sh

$GCLOUD compute scp --zone=${ZONE} secrets cybozu@${INSTANCE_NAME}:
$GCLOUD compute scp --zone=${ZONE} github-token cybozu@${INSTANCE_NAME}:
$GCLOUD compute scp --zone=${ZONE} run.sh cybozu@${INSTANCE_NAME}:
set +e
$GCLOUD compute ssh --zone=${ZONE} cybozu@${INSTANCE_NAME} --command='sudo /home/cybozu/run.sh'
RET=$?

exit $RET
