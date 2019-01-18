#!/bin/sh -e

CI_REMOTE_REPOSITORY="git@github.com:${CIRCLE_PROJECT_USERNAME}/${CIRCLE_PROJECT_REPONAME}.git"

git add -u
git commit -m "[ci skip] $(date +%Y.%m.%d)-$CIRCLE_BUILD_NUM"
git tag test-$(date +%Y.%m.%d)-$CIRCLE_BUILD_NUM
git push ${CI_REMOTE_REPOSITORY} release --tags
