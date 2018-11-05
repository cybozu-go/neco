#!/bin/sh -ex

PROJECT=neco-test
ZONE=asia-northeast1-c
SERVICE_ACCOUNT=neco-test@neco-test.iam.gserviceaccount.com
INSTANCE_NAME=${CIRCLE_PROJECT_REPONAME}-${CIRCLE_BUILD_NUM}
IMAGE_NAME=debian-9-vmx-enabled
MACHINE_TYPE=n1-standard-8
DISK_TYPE=pd-ssd
BOOT_DISK_SIZE=20GB
GCLOUD="gcloud --quiet --account ${SERVICE_ACCOUNT} --project ${PROJECT}"
ARCHIVE_SET=$1

delete_instance() {
  $GCLOUD compute instances delete ${INSTANCE_NAME} --zone ${ZONE} || true
}

# Create GCE instance
$GCLOUD compute instances create ${INSTANCE_NAME} \
  --zone ${ZONE} \
  --machine-type ${MACHINE_TYPE} \
  --image ${IMAGE_NAME} \
  --boot-disk-type ${DISK_TYPE} \
  --boot-disk-size ${BOOT_DISK_SIZE} \
  --local-ssd interface=scsi

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

# Run mtest
GOPATH=\$HOME/go
export GOPATH
PATH=/usr/local/go/bin:\$GOPATH/bin:\$PATH
export PATH

git clone https://github.com/${CIRCLE_PROJECT_USERNAME}/${CIRCLE_PROJECT_REPONAME} \$HOME/go/src/github.com/${CIRCLE_PROJECT_USERNAME}/${CIRCLE_PROJECT_REPONAME}
cd \$HOME/go/src/github.com/${CIRCLE_PROJECT_USERNAME}/${CIRCLE_PROJECT_REPONAME}
git checkout -qf ${CIRCLE_SHA1}

cd dctest
cp /assets/cybozu-ubuntu-18.04-server-cloudimg-amd64.img .
export GO111MODULE=on
make setup
exec make test
EOF
chmod +x run.sh

$GCLOUD compute scp --zone=${ZONE} run.sh cybozu@${INSTANCE_NAME}:
$GCLOUD compute ssh --zone=${ZONE} cybozu@${INSTANCE_NAME} --command='sudo /home/cybozu/run.sh'
