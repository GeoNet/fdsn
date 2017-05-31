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
BUILD_CONTAINER=golang:1.8-alpine

DOCKER_TMP=docker-build-tmp

mkdir -p $DOCKER_TMP
chmod +s $DOCKER_TMP

sudo rm -rf $DOCKER_TMP/*

VERSION='git-'`git rev-parse --short HEAD`

# The current working dir to use in GOBIN etc e.g., geonet-web
CWD=${PWD##*/}

mkdir -p ${DOCKER_TMP}/etc/ssl/certs
mkdir -p ${DOCKER_TMP}/usr/share

# Assemble common resource for ssl and timezones from the build container
docker run --rm -v ${PWD}:${PWD} ${BUILD_CONTAINER} \
    /bin/ash -c "apk add --update ca-certificates tzdata && \
    cp /etc/ssl/certs/ca-certificates.crt ${PWD}/${DOCKER_TMP}/etc/ssl/certs && \
    cp -Ra /usr/share/zoneinfo ${PWD}/${DOCKER_TMP}/usr/share"

# Assemble common resource for user.
echo "nobody:x:65534:65534:Nobody:/:" > ${DOCKER_TMP}/etc/passwd

for i in "$@"
do

    # only enable Cgo for executables that require it
    enable_cgo=0
    if [ ${i} = "fdsn-ws" ] || [ ${i} = "fdsn-holdings-consumer" ] || [ ${i} = "slink-db" ]; then
        enable_cgo=1
    fi

    # install dependencies to compile libmseed and libslink, and compile/install with the -static flag to statically link with C libs (if applicable)
	docker run -e "GOBIN=/usr/src/go/src/github.com/GeoNet/${CWD}/${DOCKER_TMP}" -e "CGO_ENABLED=${enable_cgo}" -e "GOPATH=/usr/src/go" -e "GOOS=linux" -e "BUILD=$BUILD" --rm \
		-v "$PWD":/usr/src/go/src/github.com/GeoNet/${CWD} \
		-w /usr/src/go/src/github.com/GeoNet/${CWD} ${BUILD_CONTAINER} \
		/bin/ash -c "apk add --update ca-certificates tzdata gcc make musl-dev && \
				     make -B -C /usr/src/go/src/github.com/GeoNet/${CWD}/vendor/github.com/GeoNet/kit/cvendor/libmseed && \
				     make -B -C /usr/src/go/src/github.com/GeoNet/${CWD}/vendor/github.com/GeoNet/kit/cvendor/libslink && \
		             go install -a -ldflags \"-X main.Prefix=${i}/${VERSION} -extldflags -static\" -installsuffix cgo ./cmd/${i}"

		rm -rf $DOCKER_TMP/assets
		mkdir $DOCKER_TMP/assets
		rsync --archive --quiet --ignore-missing-args  cmd/${i}/assets docker-build-tmp/

        # Add a default Dockerfile

		rm -f $DOCKER_TMP/Dockerfile

		echo "FROM scratch" > docker-build-tmp/Dockerfile
		echo "ADD ./ /" >> docker-build-tmp/Dockerfile
		echo "USER nobody" >> docker-build-tmp/Dockerfile
		echo "EXPOSE 8080" >> docker-build-tmp/Dockerfile
		echo "CMD [\"/${i}\"]" >> docker-build-tmp/Dockerfile

        # If a project specifies a Dockerfile then copy it over the top of the default file.

        rsync --ignore-missing-args cmd/${i}/Dockerfile docker-build-tmp/

		docker build -t 862640294325.dkr.ecr.ap-southeast-2.amazonaws.com/${i}:$VERSION -f docker-build-tmp/Dockerfile docker-build-tmp
		# tag latest.  Makes it easier to test with compose. 
		docker tag 862640294325.dkr.ecr.ap-southeast-2.amazonaws.com/${i}:$VERSION 862640294325.dkr.ecr.ap-southeast-2.amazonaws.com/${i}:latest

		rm -f $DOCKER_TMP/$i
done
