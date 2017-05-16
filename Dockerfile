FROM alpine:3.5

RUN apk add --no-cache ca-certificates

ADD apache_exporter /

EXPOSE 9117
ENTRYPOINT ["/apache_exporter"]