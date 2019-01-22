#!/bin/sh -e

if [ -e /var/scratch/placemat/volumes ]; then
  rm -r /var/scratch/placemat/volumes
fi

# download placemat volumes from GCS
if [ ! -e /var/scratch/placemat/volumes ]; then
  mkdir -p /var/scratch/placemat/volumes
fi
LATEST_VOLUME=$(gsutil ls -d ${GCS_SNAPSHOT_BUCKET}/volumes_* | tail -n 1)
gsutil cp -r ${LATEST_VOLUME}* /var/scratch/placemat/volumes

# start placemat
systemd-run --unit=placemat.service ${PLACEMAT} -enable-virtfs output/cluster.yml

# wait for launching VMs
sleep 10

count=$(${PMCTL} snapshot list | jq '.[] | select(. | contains("There is no snapshot available."))' | wc -l)
if [ $count -ne 0 ]; then
    echo "snapshots 'latest' can not be found."
    exit 1
fi

# load VMs
${PMCTL} snapshot load latest
