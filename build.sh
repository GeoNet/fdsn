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

VERSION='git-'$(git rev-parse --short HEAD)
ACCOUNT=$(aws sts get-caller-identity --output text --query 'Account')

# some of the projects don't have asset directory
EMPTY_DIR='empty-dir'
mkdir -p $EMPTY_DIR || :

for i in "$@"; do
  mkdir -p cmd/$i/assets
  dockerfile="Dockerfile"

  if test -f "cmd/${i}/Dockerfile"; then
    dockerfile="cmd/${i}/Dockerfile"
  else
    cat Dockerfile.tmplate > $dockerfile
    echo "CMD [\"/${i}\"]" >> $dockerfile
  fi

  docker build \
    --build-arg=BUILD="$i" \
    --build-arg=GIT_COMMIT_SHA="$VERSION" \
    --build-arg=ASSET_DIR="./cmd/$i/assets" \
    -t "${ACCOUNT}.dkr.ecr.ap-southeast-2.amazonaws.com/${i}:$VERSION" \
    -f $dockerfile .

  # tag latest.  Makes it easier to test with compose.
  docker tag \
    "${ACCOUNT}.dkr.ecr.ap-southeast-2.amazonaws.com/${i}:$VERSION" \
    "${ACCOUNT}.dkr.ecr.ap-southeast-2.amazonaws.com/${i}:latest"

done
