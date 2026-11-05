## 1.1.3 / 2026-11-05

* [CHANGE] Update prometheus/client_golang to v1.25.0, prometheus/common to v0.73.1 and prometheus/exporter-toolkit to v0.20.0 (#308, #310, #313)
* [FEATURE] `/metrics` accepts `name[]` query parameters to return only the named metrics, e.g. `/metrics?name[]=apache_up` (from client_golang v1.24.0)
* [BUGFIX] Reset `apache_info` on every scrape, so after an Apache upgrade or MPM change the old version is no longer exported alongside the new one until the exporter restarts (#115, #316)
* [BUGFIX] Parse proxy balancer `Sent`/`Rcvd` values from httpd before 2.4.35, which use M/G/T/P/E suffixes and plain byte counts. `M`/`G` values previously failed the whole scrape, and byte counts were exported as kilobytes (#192, #317)

## 1.1.2 / 2026-11-05

* [SECURITY] Rebuild with Go 1.26.10 and update golang.org/x/net to v0.62.0, golang.org/x/crypto to v0.58.0 and golang.org/x/text to v0.43.0 to resolve stdlib and dependency advisories, including GO-2026-6604, GO-2026-6612 and GO-2026-6809 (#311, #312)
* [CHANGE] Building from source now requires Go 1.26 or newer, as Go 1.25 no longer receives security fixes.
* [BUGFIX] README: fix the `--web.config.file` flag name and the Grafana dashboard link, and sync the flag help with `--help` (#314)

## 1.1.1 / 2026-06-22

* [SECURITY] Rebuild with Go 1.25.11 to resolve stdlib/toolchain advisories (CVE-2025-68121, CVE-2026-27143, GO-2026-4868, GO-2026-4337)

## 1.0.6 / 2024-01-30

* [FEATURE] Add UserAgent header

## 1.0.5 / 2023-12-22

* [BUGFIX] Add required changelog #182
