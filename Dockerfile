FROM golang:1.20.4 as base

WORKDIR /go/src/github.com/prometheus-community/apache_exporter

FROM base as builder
COPY . .
RUN go mod tidy
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

FROM quay.io/sysdig/sysdig-mini-ubi9:1.2.0 as ubi
COPY --from=builder /bin/apache_exporter /bin/apache_exporter
COPY --from=builder /user/group /user/passwd /etc/

EXPOSE     9117
USER       nobody
ENTRYPOINT [ "/bin/apache_exporter" ]