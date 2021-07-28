ARG BUILDER_IMAGE=quay.io/geonet/golang:1.15-alpine

FROM ${BUILDER_IMAGE} as builder
# Obtain ca-cert and tzdata, which we will add to the container
RUN apk add --update ca-certificates tzdata 

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

RUN echo 'nobody:x:65534:65534:Nobody:/:\' > /passwd
RUN go install -a -ldflags "-X main.Prefix=${BUILD}/${GIT_COMMIT_SHA}" /repo/cmd/fdsn-quake-consumer

FROM quay.io/geonet/alpine:3.10
RUN apk add --no-cache libxslt

COPY --from=builder /passwd /etc/passwd
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /repo/gobin/fdsn-quake-consumer /fdsn-quake-consumer

ARG ASSET_DIR
COPY ${ASSET_DIR} /assets

WORKDIR /
USER nobody
EXPOSE 8080
CMD ["/fdsn-quake-consumer"]
