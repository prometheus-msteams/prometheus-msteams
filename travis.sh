#!/bin/bash

if [[ -n ${TRAVIS_PULL_REQUEST_BRANCH} ]]; then
  export VERSION=${TRAVIS_PULL_REQUEST_BRANCH}
else
  export VERSION=${TRAVIS_TAG:-${TRAVIS_BRANCH}}
fi

echo "Building app version $VERSION"

make all VERSION=${VERSION}
make docker VERSION=${VERSION}

if [[ -n ${TRAVIS_PULL_REQUEST_BRANCH} ]]; then
  echo "Skip building docker images"
  exit 0
fi

if [[ -n ${TRAVIS_TAG} || ${TRAVIS_BRANCH} == "master" ]]; then
  make docker-tag-latest VERSION=${VERSION}

  # login to dockerhub
  make docker-hub-login
  make docker-hub-push VERSION=latest
  make docker-hub-push VERSION=${VERSION}

  # login to quay.io
  make docker-quay-login
  make docker-quay-push VERSION=latest
  make docker-quay-push VERSION=${VERSION}
fi

