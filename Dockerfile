FROM golang:1.9.2-stretch@sha256:5a94893f55b936da4e72666ca3e703d5ed12a930a7660d1b0cce22dc2dbab60e \
  as build

COPY . /go/src/github.com/Lusitaniae/apache_exporter

RUN cd /go/src/github.com/Lusitaniae/apache_exporter \
  && make \
  && sha256sum prometheus-exporter-apache

FROM quay.io/prometheus/busybox:latest

COPY --from=build /go/src/github.com/Lusitaniae/apache_exporter/prometheus-exporter-apache /bin/apache_exporter

ENTRYPOINT ["/bin/apache_exporter"]
EXPOSE     9117
