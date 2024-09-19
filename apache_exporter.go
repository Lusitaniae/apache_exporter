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

	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus"
	versioncollector "github.com/prometheus/client_golang/prometheus/collectors/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promslog"
	"github.com/prometheus/common/promslog/flag"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
	"github.com/prometheus/exporter-toolkit/web/kingpinflag"

	"github.com/Lusitaniae/apache_exporter/collector"
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
	promslogConfig := &promslog.Config{}

	// Parse flags
	flag.AddFlags(kingpin.CommandLine, promslogConfig)
	kingpin.HelpFlag.Short('h')
	kingpin.Version(version.Print("apache_exporter"))
	kingpin.Parse()
	logger := promslog.New(promslogConfig)
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
	prometheus.MustRegister(versioncollector.NewCollector("apache_exporter"))

	logger.Info("Starting apache_exporter", "version", version.Info())
	logger.Info("Build context", "build", version.BuildContext())
	logger.Info("Collect metrics from", "scrape_uri", *scrapeURI)

	// listener for the termination signals from the OS
	go func() {
		logger.Debug("Listening and waiting for graceful stop")
		sig := <-gracefulStop
		logger.Info("Caught signal. Wait 2 seconds...", "sig", sig)
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
		logger.Error(err.Error())
		os.Exit(1)
	}
	http.Handle("/", landingPage)

	server := &http.Server{}
	if err := web.ListenAndServe(server, toolkitFlags, logger); err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}
