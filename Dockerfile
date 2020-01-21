FROM golang:1.13-alpine as builder

# Source https://github.com/Lusitaniae/apache_exporter

WORKDIR /go/apache-exporter

ARG ALPINE_MIRROR=https://uk.alpinelinux.org

RUN sh -c "sed -i -e 's#http://dl-cdn.alpinelinux.org#${ALPINE_MIRROR}#g' /etc/apk/repositories" \
    && echo ${ALPINE_MIRROR}/alpine/edge/community/ >> /etc/apk/repositories \
    && apk add --no-cache --update git

COPY . /go/apache-exporter
RUN GO111MODULE=on go build -ldflags "-X main.Version=$(git describe --tags --abbrev=0) -X github.com/prometheus/common/version.Version=$(git describe --tags --abbrev=0) -X github.com/prometheus/common/version.Revision=$(git rev-parse HEAD)" -o /go/bin/apache_exporter
#RUN GO111MODULE=on go build -ldflags "-X main.Version=$(git describe --tags --abbrev=0) -X github.com/prometheus/common/version.Version=$(git describe --tags --abbrev=0) -X github.com/prometheus/common/version.Revision=$(git rev-parse HEAD) -X github.com/prometheus/common/version.BuildDate=$(date)" -o /go/bin/apache_exporter

FROM alpine

COPY --from=builder /go/bin/apache_exporter /apache_exporter

EXPOSE 9117

ENTRYPOINT ["/apache_exporter"]