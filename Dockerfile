FROM quay.io/prometheus/busybox:latest

COPY apache_exporter /bin/apache_exporter

ENTRYPOINT ["/bin/apache_exporter"]
EXPOSE     9117
