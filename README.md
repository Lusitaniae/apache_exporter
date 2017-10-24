# Apache Exporter for Prometheus [![Build Status][buildstatus]][circleci]

[![Docker Repository on Quay](https://quay.io/repository/Lusitaniae/apache-exporter/status)][quay]
[![Docker Pulls](https://img.shields.io/docker/pulls/lusotycoon/apache-exporter.svg?maxAge=604800)][hub]

Exports apache mod_status statistics via HTTP for Prometheus consumption.

With working golang environment it can be built with `go get`.  There is a [good article](https://machineperson.github.io/monitoring/2016/01/04/exporting-apache-metrics-to-prometheus.html) with build HOWTO and usage example.

Help on flags:

```
  -insecure
    	Ignore server certificate if using https. (default false)
  -log.level value
    	Only log messages with the given severity or above. Valid levels: [debug, info, warn, error, fatal, panic]. (default info)
  -scrape_uri string
    	URI to apache stub status page. (default "http://localhost/server-status/?auto")
  -telemetry.address string
    	Address on which to expose metrics. (default ":9117")
  -telemetry.endpoint string
    	Path under which to expose metrics. (default "/metrics")
  -version
    	Version of the Apache exporter.
```

Tested on Apache 2.2 and Apache 2.4.

# Using Docker

## Build

Run the following commands from the project root directory.

```
docker build -t apache_exporter .
```

## Run

```
docker run -d -p 9117:9117 apache_exporter \
  -scrape_uri "https://your.server.com/server-status/?auto"
```

## Collectors

The exporter collects a number of statistics from the server:

```
# HELP apache_accesses_total Current total apache accesses*
# TYPE apache_accesses_total counter
# HELP apache_scoreboard Apache scoreboard statuses
# TYPE apache_scoreboard gauge
# HELP apache_sent_kilobytes_total Current total kbytes sent*
# TYPE apache_sent_kilobytes_total counter
# HELP apache_cpu_load CPU Load*
# TYPE apache_cpu_load counter
# HELP apache_up Could the apache server be reached
# TYPE apache_up gauge
# HELP apache_uptime_seconds_total Current uptime in seconds*
# TYPE apache_uptime_seconds_total counter
# HELP apache_workers Apache worker statuses
# TYPE apache_workers gauge
```

Request metrics:

```
# HELP apache_requests_per_second Requests per second*
# TYPE apache_requests_per_second counter
# HELP apache_bytes_per_second Bytes per second*
# TYPE apache_bytes_per_second counter
# HELP apache_bytes_per_request Bytes per request*
# TYPE apache_bytes_per_request counter
# HELP http_request_duration_microseconds The HTTP request latencies in microseconds.
# TYPE http_request_duration_microseconds summary
# HELP http_request_size_bytes The HTTP request sizes in bytes.
# TYPE http_request_size_bytes summary
# HELP http_response_size_bytes The HTTP response sizes in bytes.
# TYPE http_response_size_bytes summary
```

Process metrics:

```
# HELP process_cpu_seconds_total Total user and system CPU time spent in seconds.
# TYPE process_cpu_seconds_total counter
# HELP process_max_fds Maximum number of open file descriptors.
# TYPE process_max_fds gauge
# HELP process_open_fds Number of open file descriptors.
# TYPE process_open_fds gauge
# HELP process_resident_memory_bytes Resident memory size in bytes.
# TYPE process_resident_memory_bytes gauge
# HELP process_start_time_seconds Start time of the process since unix epoch in seconds.
# TYPE process_start_time_seconds gauge
# HELP process_virtual_memory_bytes Virtual memory size in bytes.
# TYPE process_virtual_memory_bytes gauge
```

Metrics with '*' on the end works only when "ExtendedStatus" is set to "on" in httpd.conf (default since apache 2.3.6)

## Author

The exporter was originally created by [neezgee](https://github.com/neezgee).


[buildstatus]: https://circleci.com/gh/Lusitaniae/apache_exporter/tree/master.svg?style=shield
[quay]: https://quay.io/repository/Lusitaniae/apache-exporter
[circleci]: https://circleci.com/gh/Lusitaniae/apache_exporter
[hub]: https://hub.docker.com/r/lusotycoon/apache-exporter/
