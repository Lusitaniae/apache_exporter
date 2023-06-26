
# Apache Exporter for Prometheus [![Build Status][buildstatus]][circleci]

[![GitHub release](https://img.shields.io/github/release/Lusitaniae/apache_exporter.svg)][release]
![GitHub Downloads](https://img.shields.io/github/downloads/Lusitaniae/apache_exporter/total.svg)
[![Docker Repository on Quay](https://quay.io/repository/Lusitaniae/apache-exporter/status)][quay]
[![Docker Pulls](https://img.shields.io/docker/pulls/lusotycoon/apache-exporter.svg?maxAge=604800)][hub]

Exports apache mod_status statistics via HTTP for Prometheus consumption.

With working golang environment it can be built with `go get`.  There is a [good article](https://machineperson.github.io/monitoring/2016/01/04/exporting-apache-metrics-to-prometheus.html) with build HOWTO and usage example.

Help on flags:

<pre>
  -h, --[no-]help                Show context-sensitive help (also try
                                 --help-long and --help-man).
      --telemetry.endpoint="/metrics"
                                 Path under which to expose metrics.
      --scrape_uri="http://localhost/server-status/?auto"
                                 URI to apache stub status page.
      --host_override=""         Override for HTTP Host header; empty string for
                                 no override.
      --[no-]insecure            Ignore server certificate if using https.
      --custom_headers     Adds custom headers to the collector.
      --[no-]web.systemd-socket  Use systemd socket activation listeners instead
                                 of port listeners (Linux only).
      --web.listen-address=:9117 ...
                                 Addresses on which to expose metrics and web
                                 interface. Repeatable for multiple addresses.
      --web.config.file=""       [EXPERIMENTAL] Path to configuration file that
                                 can enable TLS or authentication.
      --log.level=info           Only log messages with the given severity or
                                 above. One of: [debug, info, warn, error]
      --log.format=logfmt        Output format of log messages. One of: [logfmt,
                                 json]
      --[no-]version             Show application version.

</pre>

Tested on Apache 2.2 and Apache 2.4.

 If your server-status page is secured by http auth, add the credentials to the scrape URL following this example:

```
http://user:password@localhost/server-status?auto
```

Override host name by runnning
```
./apache_exporter --host_override=example.com
```

# Using Docker

## Use ```compose.yml```
- Requires Docker CE
- Does not build from source, but rather pulls from registry only
1. Edit compose.yml if needed (Maybe to change server-status endpoint)
2. ```docker compose up```

### OR
## Build the compatible binary
To make sure that exporter binary created by build job is suitable to run on busybox environment, generate the binary using Makefile definition. Inside project directory run:
```
make
```
*Please be aware that binary generated using `go get` or `go build` with defaults may not work in busybox/alpine base images.*

Another option is to use a docker go container to generate the binary.

Pick an appropriate go image from https://hub.docker.com/_/golang. Inside the project directory run:
```
docker run --rm -v <project home>:/usr/local/go/src/github.com/Lusitaniae/apache_exporter -w /usr/local/go/src/github.com/Lusitaniae/apache_exporter <go docker image name> make
```

## Build image

Run the following commands from the project root directory.

```
docker build -t apache_exporter .
```

## Run

```
docker run -d -p 9117:9117 apache_exporter \
  --scrape_uri="https://your.server.com/server-status?auto"
```

## Collectors

Apache metrics:

```
# HELP apache_accesses_total Current total apache accesses (*)
# TYPE apache_accesses_total counter
# HELP apache_scoreboard Apache scoreboard statuses
# TYPE apache_scoreboard gauge
# HELP apache_sent_kilobytes_total Current total kbytes sent (*)
# TYPE apache_sent_kilobytes_total counter
# HELP apache_cpu_load CPU Load (*)
# TYPE apache_cpu_load gauge
# HELP apache_up Could the apache server be reached
# TYPE apache_up gauge
# HELP apache_uptime_seconds_total Current uptime in seconds (*)
# TYPE apache_uptime_seconds_total counter
# HELP apache_workers Apache worker statuses
# TYPE apache_workers gauge
# HELP apache_version Apache server version
# TYPE apache_version gauge
# HELP apache_duration_ms_total Total duration of all registered requests
# TYPE apache_duration_ms_total gauge
```

Metrics marked '(*)' are only available if ExtendedStatus is On in apache webserver configuration. In version 2.3.6, loading mod_status will toggle ExtendedStatus On by default.

## FAQ

Q. Can I change the Dockerfile?

A. In short no, it's not meant for end users. It's part of the CI/CD pipeline in which promu is cross building the exporter for all architectures and packaging them into a Docker image.

Q. Can I run this exporter on different architectures (ARM)?

A. This exporter is cross compiled to all architectures using [promu](https://github.com/prometheus/promu) by running `promu crossbuild`. You can find the resulting artifacts in the release page (Github) or docker images in [Quay](https://quay.io/repository/Lusitaniae/apache-exporter) or [Docker](https://hub.docker.com/r/lusotycoon/apache-exporter/).

Q. Is there a Grafana dashboard for this exporter?

A. There's a 3rd party dashboard [here](https://grafana.com/dashboards/3894) which seems to work.
Also [monitoring-mixin](https://monitoring.mixins.dev/) (dashboard+prometheus alerts) is available [here](https://github.com/grafana/jsonnet-libs/tree/master/apache-http-mixin).

Q. Can you add additional metrics such as reqpersec, bytespersec and bytesperreq?

A. In line with the [best practices](https://prometheus.io/docs/instrumenting/writing_exporters/#drop-less-useful-statistics), the exporter only provides the totals and you should derive rates using [PromQL](https://prometheus.io/docs/prometheus/latest/querying/basics/).

>   Like that:
>   - `ReqPerSec` : `rate(apache_accesses_total[5m])`
>   - `BytesPerSec`: `rate(apache_sent_kilobytes_total[5m])`
>   - `BytesPerReq`: BytesPerSec / ReqPerSec

Q. Can I monitor multiple Apache instances?

A. In line with the [best practices](https://prometheus.io/docs/instrumenting/writing_exporters/#deployment), the answer is no. *Each process being monitored should be accompanied by **one** exporter*.

We suggest automating configuration and deployment using your favorite tools, e.g. Ansible/Chef/Kubernetes.

Q. Its not working! Apache_up shows as 0

A. When apache_up reports 0 it means the exporter is running however it is not able to connect to Apache.

Do you have this (or similar) configuration uncommented in your Apache instance?
```
<Location "/server-status">
    SetHandler server-status
    Require host example.com
</Location>
```
As documented at
https://httpd.apache.org/docs/2.4/mod/mod_status.html

Are you able to see the stats, if you run this from the Apache instance?

`curl localhost/server-status?auto`

If you run the exporter manually, do you see any errors?

`./apache_exporter`

Please include all this information if you still have issues when creating an issue.

Q. There seem to be missing metrics! I can only see apache_up and apache_cpuload.

A. Make sure that you add `?auto` at the end of the scrape_uri.



## TLS and basic authentication

Apache Exporter supports TLS and basic authentication. This enables better
control of the various HTTP endpoints.

To use TLS and/or basic authentication, you need to pass a configuration file
using the `--web.config` parameter (see above at help on flags). The format of the file is described
[in the exporter-toolkit repository](https://github.com/prometheus/exporter-toolkit/blob/master/docs/web-configuration.md).

Note that the TLS and basic authentication settings affect all HTTP endpoints:
/metrics for scraping, /probe for probing, and the web UI.


## Author

The exporter was originally created by [neezgee](https://github.com/neezgee).


[buildstatus]: https://circleci.com/gh/Lusitaniae/apache_exporter/tree/master.svg?style=shield
[quay]: https://quay.io/repository/Lusitaniae/apache-exporter
[circleci]: https://circleci.com/gh/Lusitaniae/apache_exporter
[hub]: https://hub.docker.com/r/lusotycoon/apache-exporter/
[release]: https://github.com/Lusitaniae/apache_exporter/releases/latest
