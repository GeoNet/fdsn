FROM alpine:3.9

RUN apk add --no-cache libxslt

ADD ./ /

WORKDIR /
USER nobody
EXPOSE 8080
CMD ["/fdsn-quake-consumer"]
