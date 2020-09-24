#!/bin/bash -e

# Builds and pushes Docker images for the arg list.
#
# usage: ./build-push.sh project [project]

./build.sh $@

ACCOUNT=$(aws sts get-caller-identity --output text --query 'Account')
VERSION='git-'`git rev-parse --short HEAD`
eval $(aws ecr get-login --no-include-email --region ap-southeast-2)

for i in "$@"
do
		docker push ${ACCOUNT}.dkr.ecr.ap-southeast-2.amazonaws.com/${i}:$VERSION 
		docker push ${ACCOUNT}.dkr.ecr.ap-southeast-2.amazonaws.com/${i}:latest
done
