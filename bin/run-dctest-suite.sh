#!/bin/sh -ex

. $(dirname $0)/env
SUITE_NAME=$1
TAG_NAME=$2

# Run data center test
cat >run.sh <<EOF
#!/bin/sh -e

GOPATH=\$HOME/go
export GOPATH
PATH=/usr/local/go/bin:\$GOPATH/bin:\$PATH
export PATH

# Run dctest
cd \$GOPATH/src/github.com/cybozu-go/neco/dctest
export GO111MODULE=on
exec make test TAGS=${TAG_NAME} SUITE=${SUITE_NAME}
EOF
chmod +x run.sh

$GCLOUD compute scp --zone=${ZONE} run.sh cybozu@${INSTANCE_NAME}:
set +e
$GCLOUD compute ssh --zone=${ZONE} cybozu@${INSTANCE_NAME} --command='sudo -H /home/cybozu/run.sh'
exit $?
