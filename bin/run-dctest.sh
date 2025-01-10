#!/bin/sh -ex

. $(dirname $0)/env
SUITE_NAME=$1
TAG_NAME=$2
DATACENTER=$3
MENU=$4

# Convert git refs to the acceptable label value.
# - The value consists of lowercase letters, numeric characters, underscores, and dashes.
# - The value is shorter than 64 characters.
GIT_REFNAME=${CIRCLE_BRANCH}${CIRCLE_TAG}
REF_VALUE=$(echo ${GIT_REFNAME} | tr '[A-Z]' '[a-z]' | sed -e "s/[^-a-zA-Z0-9]\+/_/g" | cut -c-63)

# GCE instance labels
LABEL_REPO=repo=${CIRCLE_PROJECT_REPONAME}
LABEL_JOB=job=${CIRCLE_JOB}
LABEL_REF=ref=${REF_VALUE}

# Set state label to old instances
CANCELED_LIST=$($GCLOUD compute instances list --project neco-test --filter="labels.${LABEL_REPO} AND labels.${LABEL_REF} AND labels.${LABEL_JOB}" --format "value(name)")
for CANCELED in ${CANCELED_LIST}; do
  $GCLOUD compute instances add-labels ${CANCELED} --zone=${ZONE} --labels=state=canceled || true
done

# Create GCE instance
LOCAL_SSD=
for i in $(seq 1 ${LOCAL_SSD_COUNT}); do
  LOCAL_SSD="--local-ssd interface=nvme ${LOCAL_SSD}"
done
$GCLOUD compute instances delete ${INSTANCE_NAME} --zone=${ZONE} --quiet || true
$GCLOUD compute instances create ${INSTANCE_NAME} \
  --zone ${ZONE} \
  --machine-type ${MACHINE_TYPE} \
  --image vmx-enabled \
  --boot-disk-type ${DISK_TYPE} \
  --boot-disk-size ${BOOT_DISK_SIZE} \
  ${LOCAL_SSD} \
  --labels=${LABEL_REPO},${LABEL_REF},${LABEL_JOB}

# Wait for boot of GCE instance
for i in $(seq 300); do
  if $GCLOUD compute ssh --zone=${ZONE} cybozu@${INSTANCE_NAME} --command=date 2>/dev/null; then
    break
  fi
  sleep 1
done

# Prepare script for running data center test
cat >run.sh <<EOF
#!/bin/sh -ex

# mkfs and mount local SSD on /var/scratch
NUM_DEVICES=\$(ls /dev/nvme0n*|wc -l)
mdadm --create /dev/md0 -l stripe --raid-devices=\${NUM_DEVICES} /dev/nvme0n*
mkfs -t ext4 -F /dev/md0
mkdir -p /var/scratch
mount -t ext4 /dev/md0 /var/scratch
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
rm -f /home/cybozu/neco.tgz
if [ -f download.tgz ]; then
  tar xzf download.tgz
  rm -f download.tgz
fi

# Prepare files
mkdir -p \${NECO_DIR}/installer/build
cp /assets/ubuntu-*-server-cloudimg-amd64.img \${NECO_DIR}/installer/build
mkdir -p \${NECO_DIR}/dctest/output
cp /assets/flatcar_production_qemu_image.img \${NECO_DIR}/dctest/output

# Check out released sources which are used in the specified datacenter
if [ -n "${DATACENTER}" ]; then
  go install ./pkg/find-installed-release
  RELEASE=\$(find-installed-release ${DATACENTER})
  git worktree add \${TEMP_DIR} release-\$RELEASE
  cd \${TEMP_DIR}

  # Prepare files
  mkdir -p \${TEMP_DIR}/installer/build
  cp /assets/ubuntu-*-server-cloudimg-amd64.img \${TEMP_DIR}/installer/build
  mkdir -p \${TEMP_DIR}/dctest/output
  cp /assets/flatcar_production_qemu_image.img \${TEMP_DIR}/dctest/output
  cp \${NECO_DIR}/github-token \${TEMP_DIR}
fi

# Run dctest
cd dctest
make setup
make placemat TAGS=${TAG_NAME} MENU_ARG=${MENU}
sleep 3
exec make test TAGS=${TAG_NAME} SUITE=${SUITE_NAME} DATACENTER=${DATACENTER}
EOF
chmod +x run.sh

# Send files to GCE instance
# The latter half (after '||') is workaround for "tar: .: file changed as we read it"
tar czf /tmp/neco.tgz . || [ $? = 1 ]
$GCLOUD compute scp --zone=${ZONE} /tmp/neco.tgz cybozu@${INSTANCE_NAME}:
$GCLOUD compute scp --zone=${ZONE} run.sh cybozu@${INSTANCE_NAME}:

# Run data center test on GCE instance
set +e
$GCLOUD compute ssh --zone=${ZONE} cybozu@${INSTANCE_NAME} --command='sudo -H /home/cybozu/run.sh'
exit $?
