ARG BUILDER_IMAGE=ghcr.io/geonet/base-images/golang:1.23.5-alpine3.21

FROM ${BUILDER_IMAGE} as builder
# Obtain ca-cert and tzdata, which we will add to the container
# Git commit SHA
ARG GIT_COMMIT_SHA
ARG BUILD
ADD ./ /repo
WORKDIR /repo

# Set a bunch of go env flags
ENV GOBIN /repo/gobin
ENV GOPATH /usr/src/go
ENV GOFLAGS -mod=vendor
ENV GOOS linux
ENV GOARCH amd64
ENV CGO_ENABLED 0
RUN go install -a -ldflags "-X main.Prefix=${BUILD}/${GIT_COMMIT_SHA}" /repo/cmd/fdsn-quake-consumer

FROM ghcr.io/geonet/base-images/alpine-xslt:3.18
COPY --from=builder /repo/gobin/fdsn-quake-consumer /fdsn-quake-consumer
ARG ASSET_DIR
COPY ${ASSET_DIR} /assets
WORKDIR /
USER nobody
EXPOSE 8080
CMD ["/fdsn-quake-consumer"]
