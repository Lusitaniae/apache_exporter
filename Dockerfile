FROM golang:latest as builder

WORKDIR /go
COPY . .
RUN go get -d -v .
RUN go build -o ./apache_exporter .

FROM quay.io/prometheus/busybox:latest

COPY --from=builder /go/apache_exporter /bin/apache_exporter

ENTRYPOINT ["/bin/apache_exporter"]
EXPOSE     9117
