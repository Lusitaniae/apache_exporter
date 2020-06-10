# Go container to build apache-exporter
FROM golang:1.14.4 as go

WORKDIR /exporter

COPY . .

RUN make

# Second stage, conatiner with the exporter
FROM busybox:1.31.1

COPY --from=go /exporter/apache_exporter /bin/apache_exporter

EXPOSE 9117

ENTRYPOINT ["/bin/apache_exporter"]
