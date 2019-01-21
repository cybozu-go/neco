#!/bin/sh

VOLUMES=$(gsutil ls -d gs://neco-public/snapshots/volumes_*)
NOW=$(date "+%Y%m%d%H%M%S")
NODES=$(${PMCTL} node list)

${PMCTL} snapshot save latest

for node in ${NODES}; do
  ${PMCTL} node action stop ${node}
done

gsutil cp -r ${PLACEMAT_DATADIR}/volumes ${GCS_SNAPSHOT_BUCKET}/volumes_${NOW}

for v in ${VOLUMES}; do
  gsutil rm $v
done
