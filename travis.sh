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

echo ${DOCKER_PASSWORD} | docker login --password-stdin -u ${DOCKER_USER}

if [[ -n ${TRAVIS_TAG} || ${TRAVIS_BRANCH} == "master" ]]; then
  make docker-tag-latest VERSION=${VERSION}
  make docker-push VERSION=latest
  make docker-push VERSION=${VERSION}
fi

