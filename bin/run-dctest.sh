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
$GCLOUD compute instances delete ${INSTANCE_NAME} --zone=${ZONE} --quiet || true
$GCLOUD compute instances create ${INSTANCE_NAME} \
  --zone ${ZONE} \
  --machine-type ${MACHINE_TYPE} \
  --image vmx-enabled \
  --boot-disk-type ${DISK_TYPE} \
  --boot-disk-size ${BOOT_DISK_SIZE} \
  --local-ssd interface=nvme \
  --local-ssd interface=nvme \
  --local-ssd interface=nvme \
  --local-ssd interface=nvme \
  --local-ssd interface=nvme \
  --local-ssd interface=nvme \
  --local-ssd interface=nvme \
  --local-ssd interface=nvme \
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
mkfs -t ext4 -F /dev/nvme0n1
mkdir -p /var/scratch
mount -t ext4 /dev/nvme0n1 /var/scratch
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
if [ -f download.tgz ]; then
  tar xzf download.tgz
fi

# Prepare files
cp \${NECO_DIR}/github-token \${NECO_DIR}/dctest/
mkdir -p \${NECO_DIR}/installer/build
cp /assets/ubuntu-20.04-server-cloudimg-amd64.img \${NECO_DIR}/installer/build

# Check out released sources which are used in the specified datacenter
if [ -n "${DATACENTER}" ]; then
  go install ./pkg/find-installed-release
  RELEASE=\$(find-installed-release ${DATACENTER})
  git worktree add \${TEMP_DIR} release-\$RELEASE
  cd \${TEMP_DIR}

  # Prepare files
  cp \${NECO_DIR}/github-token \${TEMP_DIR}/dctest/
  mkdir -p \${NECO_DIR}/installer/build
  cp /assets/ubuntu-20.04-server-cloudimg-amd64.img \${NECO_DIR}/installer/build
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
tar czf /tmp/neco.tgz .
$GCLOUD compute scp --zone=${ZONE} /tmp/neco.tgz cybozu@${INSTANCE_NAME}:
$GCLOUD compute scp --zone=${ZONE} run.sh cybozu@${INSTANCE_NAME}:

# Run data center test on GCE instance
set +e
$GCLOUD compute ssh --zone=${ZONE} cybozu@${INSTANCE_NAME} --command='sudo -H /home/cybozu/run.sh'
exit $?
