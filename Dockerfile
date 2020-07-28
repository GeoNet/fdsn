ARG BUILDER_IMAGE=quay.io/geonet/golang:1.14-alpine
ARG RUNNER_IMAGE=quay.io/geonet/alpine:3.10
ARG RUN_USER=nobody
# Only support image based on AlpineLinux
FROM ${BUILDER_IMAGE} as builder

# Obtain ca-cert and tzdata, which we will add to the container
RUN apk add --update ca-certificates tzdata

# Project to build
ARG BUILD

# Git commit SHA
ARG GIT_COMMIT_SHA

COPY ./ /repo

WORKDIR /repo

# Make C libraries
RUN apk add --update gcc make musl-dev && \
    make -B -C /repo/vendor/github.com/GeoNet/kit/cvendor/libmseed && \
    make -B -C /repo/vendor/github.com/GeoNet/kit/cvendor/libslink

# Set a bunch of go env flags
ARG CGO=1
ENV GOBIN /repo/gobin
ENV GOPATH /usr/src/go
ENV GOFLAGS -mod=vendor
ENV CGO_ENABLED ${CGO}
ENV GOOS linux
ENV GOARCH amd64

RUN echo 'nobody:x:65534:65534:Nobody:/:\' > /passwd
RUN go install -a -installsuffix cgo -ldflags "-X main.Prefix=${BUILD}/${GIT_COMMIT_SHA} -extldflags -static" /repo/cmd/${BUILD}

FROM ${RUNNER_IMAGE}
# Export a port, default to 8080
ARG EXPOSE_PORT=8080
EXPOSE $EXPOSE_PORT
# Asset directory to copy to /assets
# Add common resource for ssl and timezones from the build container
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/
# Create a nobody user
COPY --from=builder /passwd /etc/passwd
# Same ARG as before
ARG BUILD
# Need to make this an env for it to be interpolated by the shell
ENV BUILD_BIN=${BUILD}
# We have to make our binary have a fixed name, otherwise, we cannot run it without a shell
COPY --from=builder /repo/gobin/$BUILD /app
# Copy the assets
ARG ASSET_DIR
COPY ${ASSET_DIR} /assets
USER ${RUN_USER}
CMD ["/app"]