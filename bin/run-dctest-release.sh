#!/bin/sh -ex

. $(dirname $0)/env

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

# Run dctest
GOPATH=\$HOME/go
export GOPATH
PATH=/usr/local/go/bin:\$GOPATH/bin:\$PATH
export PATH

cd /home/cybozu/${CIRCLE_PROJECT_REPONAME}

cd dctest
cp /home/cybozu/${CIRCLE_PROJECT_REPONAME}/secrets .
cp /assets/cybozu-ubuntu-18.04-server-cloudimg-amd64.img .
export GO111MODULE=on
make setup
exec make MENU=highcpu-menu.yml test
EOF
chmod +x run.sh

$GCLOUD compute scp --zone=${ZONE} --recurse . cybozu@${INSTANCE_NAME}:${CIRCLE_PROJECT_REPONAME}
$GCLOUD compute ssh --zone=${ZONE} cybozu@${INSTANCE_NAME} --command='sudo /home/cybozu/${CIRCLE_PROJECT_REPONAME}/run.sh'
