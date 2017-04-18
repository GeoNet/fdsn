FROM alpine:3.4

RUN apk add --no-cache libxslt ca-certificates tzdata

ADD ./ /

WORKDIR /
USER nobody
EXPOSE 8080
CMD ["/fdsn-ws"]
