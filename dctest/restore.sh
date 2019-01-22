#!/bin/sh -e

if [ -e /var/scratch/placemat/volumes ]; then
  sudo rm -r /var/scratch/placemat/volumes
fi

# download placemat volumes from GCS
if [ ! -e /var/scratch/placemat/volumes ]; then
  sudo mkdir -p /var/scratch/placemat/volumes
fi
LATEST_VOLUME=$(gsutil ls -d ${GCS_SNAPSHOT_BUCKET}/volumes_* | tail -n 1)
sudo gsutil cp -r ${LATEST_VOLUME}* /var/scratch/placemat/volumes

# start placemat
sudo -b ${PLACEMAT} -enable-virtfs output/cluster.yml

# wait for launching VMs
for i in $(seq 10); do
  if ${PMCTL} node list | wc -l | test $(cat -) -eq 10 2> /dev/null; then
    break
  fi
  sleep 1
done

count=$(${PMCTL} snapshot list | jq '.[] | select(. | contains("latest"))' | wc -l)
if [ $count -ne 10 ]; then
    echo "snapshots 'latest' can not be found."
    exit 1
fi

# load VMs
${PMCTL} snapshot load latest
