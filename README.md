
# Apache Exporter for Prometheus [![Build Status][buildstatus]][circleci]

[![GitHub release](https://img.shields.io/github/release/Lusitaniae/apache_exporter.svg)][release]
![GitHub Downloads](https://img.shields.io/github/downloads/Lusitaniae/apache_exporter/total.svg)
[![Docker Repository on Quay](https://quay.io/repository/Lusitaniae/apache-exporter/status)][quay]
[![Docker Pulls](https://img.shields.io/docker/pulls/lusotycoon/apache-exporter.svg?maxAge=604800)][hub]

Exports apache mod_status statistics via HTTP for Prometheus consumption.

With working golang environment it can be built with `go get`.  There is a [good article](https://machineperson.github.io/monitoring/2016/01/04/exporting-apache-metrics-to-prometheus.html) with build HOWTO and usage example.

Help on flags:

<pre>
  -h, --help
      --telemetry.address=":9117"
                          Address on which to expose metrics.
      --telemetry.endpoint="/metrics"
                          Path under which to expose metrics.
      --scrape_uri="http://localhost/server-status/?auto"
                          URI to apache stub status page.
      --host_override=""  Override for HTTP Host header; empty string for no override.
      --insecure          Ignore server certificate if using https.
      --log.level="info"  Only log messages with the given severity or above. Valid levels: [debug, info, warn, error, fatal]
      --log.format="logger:stderr"
                          Set the log target and format. Example: "logger:syslog?appname=bob&local=7" or "logger:stdout?json=true"
      --version           Show application version.
</pre>

Default values can be changed by environment variables:
<pre>
TELEMETRY_ADDRESS
TELEMETRY_ENDPOINT
SCRAPE_URI
HOST_OVERRIDE
INSECURE (true or false)
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
  --scrape_uri="https://your.server.com/server-status/?auto"
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

Q. Is there a Grafana dashboard for this exporter?

A. There's a 3rd party dashboard [here](https://grafana.com/dashboards/3894) which seems to work. 

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

## Author

The exporter was originally created by [neezgee](https://github.com/neezgee).


[buildstatus]: https://circleci.com/gh/Lusitaniae/apache_exporter/tree/master.svg?style=shield
[quay]: https://quay.io/repository/Lusitaniae/apache-exporter
[circleci]: https://circleci.com/gh/Lusitaniae/apache_exporter
[hub]: https://hub.docker.com/r/lusotycoon/apache-exporter/
[release]: https://github.com/Lusitaniae/apache_exporter/releases/latest
