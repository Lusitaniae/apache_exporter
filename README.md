This repository is obsolete. The exporter is now maintened here: https://github.com/Lusitaniae/apache_exporter

# Apache Exporter for Prometheus

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
