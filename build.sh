#!/bin/bash -e

# Builds Docker images for the arg list.  These must be project directories
# where this script is executed.
#
# Builds a statically linked executable and adds it to the container.
# Adds the assets dir from each project to the container e.g., origin/assets
# It is not an error for the assets dir to not exist.
# Any assets needed by the application should be read from the assets dir
# relative to the executable. 
#
# usage: ./build.sh project [project]
if [ $# -eq 0 ]; then
  echo Error: please supply a project to build. Usage: ./build.sh project [project]
  exit 1
fi

# code will be compiled in this container
BUILDER_IMAGE='quay.io/geonet/golang:1.13.1-alpine'
RUNNER_IMAGE='quay.io/geonet/alpine:3.10'


VERSION='git-'$(git rev-parse --short HEAD)
ACCOUNT=$(aws sts get-caller-identity --output text --query 'Account')
CGO_ENABLED=0

# some of the projects don't have asset directory
EMPTY_DIR='empty-dir'
mkdir -p $EMPTY_DIR || :

for i in "$@"; do
  if [ ${i} = "fdsn-ws" ] || [ ${i} = "fdsn-holdings-consumer" ] || [ ${i} = "fdsn-slink-db" ]; then
    CGO_ENABLED=1
  fi

  ASSET_DIR=$EMPTY_DIR
  if [[ -d "./cmd/${i}/assets" ]]; then
    ASSET_DIR="./cmd/${i}/assets"
  fi

  docker build \
    --build-arg=BUILD="$i" \
    --build-arg=RUNNER_IMAGE="$RUNNER_IMAGE" \
    --build-arg=BUILDER_IMAGE="$BUILDER_IMAGE" \
    --build-arg=GIT_COMMIT_SHA="$VERSION" \
    --build-arg=ASSET_DIR="$ASSET_DIR" \
    --build-arg=CGO_ENABLED="${CGO_ENABLED}" \
    -t "${ACCOUNT}.dkr.ecr.ap-southeast-2.amazonaws.com/${i}:$VERSION" \
    -f "Dockerfile" .

  # tag latest.  Makes it easier to test with compose.
  docker tag \
    "${ACCOUNT}.dkr.ecr.ap-southeast-2.amazonaws.com/${i}:$VERSION" \
    "${ACCOUNT}.dkr.ecr.ap-southeast-2.amazonaws.com/${i}:latest"

done
