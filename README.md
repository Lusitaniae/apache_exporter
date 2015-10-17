# Apache Exporter for Prometheus

Exports apache mod_status statistics via HTTP for Prometheus consumption.

With working golang environment it can be build with `go get`.

Help on flags:

```
/apache_exporter --help
Usage of /home/ruslan/go/bin/apache_exporter:
  -insecure
    	Ignore server certificate if using https (default true)
  -log.level value
    	Only log messages with the given severity or above. Valid levels: [debug, info, warn, error, fatal, panic]. (default info)
  -scrape_uri string
    	URI to apache stub status page (default "http://localhost/server-status/?auto")
  -telemetry.address string
    	Address on which to expose metrics. (default ":9114")
  -telemetry.endpoint string
    	Path under which to expose metrics. (default "/metrics")
```
