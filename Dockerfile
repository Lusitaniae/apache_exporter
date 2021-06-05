ARG ARCH="amd64"
ARG OS="linux"
FROM quay.io/prometheus/busybox-${OS}-${ARCH}:latest
LABEL maintainer="https://github.com/Lusitaniae, https://github.com/roidelapluie"

ARG ARCH="amd64"
ARG OS="linux"
COPY .build/${OS}-${ARCH}/apache_exporter /bin/apache_exporter

EXPOSE      9117
USER        nobody
ENTRYPOINT  [ "/bin/apache_exporter" ]