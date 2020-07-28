#!/bin/bash -eu

# Builds and pushes Docker images for the arg list.
#
# usage: ./build-push.sh project [project]

./build.sh "$@"

VERSION='git-'$(git rev-parse --short HEAD)

for i in "$@"; do
  docker push "862640294325.dkr.ecr.ap-southeast-2.amazonaws.com/${i}:$VERSION"
  docker push "862640294325.dkr.ecr.ap-southeast-2.amazonaws.com/${i}:latest"
done
