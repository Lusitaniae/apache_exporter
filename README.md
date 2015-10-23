# Apache Exporter for Prometheus

Exports apache mod_status statistics via HTTP for Prometheus consumption.

With working golang environment it can be built with `go get`.

Help on flags:

```
  -insecure
    	Ignore server certificate if using https (default true)
  -log.level value
    	Only log messages with the given severity or above. Valid levels: [debug, info, warn, error, fatal, panic]. (default info)
  -scrape_uri string
    	URI to apache stub status page (default "http://localhost/server-status/?auto")
  -telemetry.address string
    	Address on which to expose metrics. (default ":9117")
  -telemetry.endpoint string
    	Path under which to expose metrics. (default "/metrics")
```

Tested on Apache 2.2 and Apache 2.4.
