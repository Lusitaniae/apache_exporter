FROM golang:alpine

RUN apk add --no-cache ca-certificates git
RUN go get github.com/neezgee/apache_exporter

EXPOSE 9117
ENTRYPOINT ["/go/bin/apache_exporter"]