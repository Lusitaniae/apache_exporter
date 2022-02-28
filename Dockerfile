FROM golang:1.17.3 as base

WORKDIR /go/src/github.com/prometheus-community/apache_exporter

FROM base as builder
COPY go.mod go.sum ./
RUN go mod download
COPY apache_exporter.go apache_exporter.go
COPY .promu.yml .promu.yml
COPY Makefile Makefile
COPY Makefile.common Makefile.common
RUN make build
RUN cp apache_exporter /bin/apache_exporter

RUN mkdir /user && \
    echo 'nobody:x:65534:65534:nobody:/:' > /user/passwd && \
    echo 'nobody:x:65534:' > /user/group


FROM scratch as scratch
COPY --from=builder /bin/apache_exporter /bin/apache_exporter
COPY --from=builder /user/group /user/passwd /etc/

EXPOSE      9117
USER        nobody
ENTRYPOINT  [ "/bin/apache_exporter" ]

FROM quay.io/sysdig/sysdig-mini-ubi:1.2.9 as ubi
COPY --from=builder /bin/apache_exporter /bin/apache_exporter
COPY --from=builder /user/group /user/passwd /etc/

EXPOSE     9117
USER       nobody
ENTRYPOINT [ "/bin/apache_exporter" ]