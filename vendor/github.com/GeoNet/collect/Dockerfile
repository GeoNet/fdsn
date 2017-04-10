FROM golang:1.8-alpine
MAINTAINER "Mark Chadwick" <m.chadwick@gns.cri.nz>
ARG APP
ARG VERSION

ADD . /go/src/github.com/GeoNet/collect
# shakenz config needs .pb file to be in pwd
ADD shakenz-slink/shakenz-slink.pb /go/shakenz-slink.pb

RUN apk --update add --no-cache make gcc libc-dev && \
        cd /go/src/github.com/GeoNet/collect/cvendor/libmseed/ && make -B && \
        cd /go/src/github.com/GeoNet/collect/cvendor/libslink/ && make -B && \
        cd /go/src/github.com/GeoNet/collect && \
        go install --ldflags '-extldflags "-static" -X main.Prefix=${APP}/${VERSION}' ./...

#ENTRYPOINT ["/go/bin/shakenz"]
