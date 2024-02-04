// Copyright (c) 2015 neezgee
//
// Licensed under the MIT license: https://opensource.org/licenses/MIT
// Permission is granted to use, copy, modify, and redistribute the work.
// Full license information available in the project LICENSE file.
//

package main

import (
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Lusitaniae/apache_exporter/collector"
	"github.com/alecthomas/kingpin/v2"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
	"github.com/prometheus/exporter-toolkit/web/kingpinflag"
)

var (
	metricsEndpoint = kingpin.Flag("telemetry.endpoint", "Path under which to expose metrics.").Default("/metrics").String()
	scrapeURI       = kingpin.Flag("scrape_uri", "URI to apache stub status page.").Default("http://localhost/server-status/?auto").String()
	hostOverride    = kingpin.Flag("host_override", "Override for HTTP Host header; empty string for no override.").Default("").String()
	insecure        = kingpin.Flag("insecure", "Ignore server certificate if using https.").Bool()
	toolkitFlags    = kingpinflag.AddFlags(kingpin.CommandLine, ":9117")
	gracefulStop    = make(chan os.Signal, 1)
	customHeaders   = kingpin.Flag("custom_headers", "Adds custom headers to the collector.").StringMap()
)

func main() {
	promlogConfig := &promlog.Config{}

	// Parse flags
	flag.AddFlags(kingpin.CommandLine, promlogConfig)
	kingpin.HelpFlag.Short('h')
	kingpin.Version(version.Print("apache_exporter"))
	kingpin.Parse()
	logger := promlog.New(promlogConfig)
	// listen to termination signals from the OS
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)
	signal.Notify(gracefulStop, syscall.SIGHUP)
	signal.Notify(gracefulStop, syscall.SIGQUIT)

	config := &collector.Config{
		ScrapeURI:     *scrapeURI,
		HostOverride:  *hostOverride,
		Insecure:      *insecure,
		CustomHeaders: *customHeaders,
	}

	exporter := collector.NewExporter(logger, config)
	prometheus.MustRegister(exporter)
	prometheus.MustRegister(version.NewCollector("apache_exporter"))

	level.Info(logger).Log("msg", "Starting apache_exporter", "version", version.Info())
	level.Info(logger).Log("msg", "Build context", "build", version.BuildContext())
	level.Info(logger).Log("msg", "Collect from: ", "scrape_uri", *scrapeURI)

	// listener for the termination signals from the OS
	go func() {
		level.Info(logger).Log("msg", "listening and wait for graceful stop")
		sig := <-gracefulStop
		level.Info(logger).Log("msg", "caught sig: %+v. Wait 2 seconds...", "sig", sig)
		time.Sleep(2 * time.Second)
		os.Exit(0)
	}()

	http.Handle(*metricsEndpoint, promhttp.Handler())

	landingConfig := web.LandingConfig{
		Name:        "Apache Exporter",
		Description: "Prometheus exporter for Apache HTTP server metrics",
		Version:     version.Info(),
		Links: []web.LandingLinks{
			{
				Address: *metricsEndpoint,
				Text:    "Metrics",
			},
		},
	}
	landingPage, err := web.NewLandingPage(landingConfig)
	if err != nil {
		level.Error(logger).Log("err", err)
		os.Exit(1)
	}
	http.Handle("/", landingPage)

	server := &http.Server{}
	if err := web.ListenAndServe(server, toolkitFlags, logger); err != nil {
		level.Error(logger).Log("err", err)
		os.Exit(1)
	}
}
