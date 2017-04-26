FROM alpine:3.5

RUN apk add --no-cache libxslt

ADD ./ /

WORKDIR /
USER nobody
EXPOSE 8080
CMD ["/fdsn-ws"]
