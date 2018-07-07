#!/bin/bash
set -e
set -o pipefail

# Install docker client
# set -x
# VER="17.03.0-ce"
# mkdir -p /tmp
# curl -L -o /tmp/docker-$VER.tgz https://get.docker.com/builds/Linux/x86_64/docker-$VER.tgz
# tar -xz -C /tmp -f /tmp/docker-$VER.tgz
# mv /tmp/docker/* /usr/bin -f

# push to gcloud docker registry
IMAGE=asia.gcr.io/$GCLOUD_PROJECT_ID/go-resizer-vips
DOCKERCTX=.

if [[ ! -z "${CIRCLE_TAG}" ]]; then
    TAG=$(echo "$CIRCLE_TAG" | sed 's/^v//g')
    SHA=$(echo ${CIRCLE_SHA1:0:8})
    echo "push ${TAG} ${SHA}"
    docker build -t $IMAGE:$TAG -t $IMAGE:$SHA $DOCKERCTX
    docker login -u _json_key -p "$GCLOUD_SERVICE_KEY" https://asia.gcr.io
    docker push $IMAGE:$TAG
    docker push $IMAGE:$SHA
fi
if [[ "${CIRCLE_BRANCH}" == "master" && -z "${CIRCLE_TAG}" ]]; then
    SHA=$(echo ${CIRCLE_SHA1:0:8})
    echo "push latest ${SHA}"
    docker build -t $IMAGE:latest -t $IMAGE:$SHA $DOCKERCTX
    docker login -u _json_key -p "$GCLOUD_SERVICE_KEY" https://asia.gcr.io
    docker push $IMAGE:latest
    docker push $IMAGE:$SHA
fi