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
	kingpin "github.com/alecthomas/kingpin/v2"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
)

var (
	listeningAddress = kingpin.Flag("telemetry.address", "Address on which to expose metrics.").Default(":9117").String()
	metricsEndpoint  = kingpin.Flag("telemetry.endpoint", "Path under which to expose metrics.").Default("/metrics").String()
	scrapeURI        = kingpin.Flag("scrape_uri", "URI to apache stub status page.").Default("http://localhost/server-status/?auto").String()
	hostOverride     = kingpin.Flag("host_override", "Override for HTTP Host header; empty string for no override.").Default("").String()
	insecure         = kingpin.Flag("insecure", "Ignore server certificate if using https.").Bool()
	configFile       = kingpin.Flag("web.config", "Path to config yaml file that can enable TLS or authentication.").Default("").String()
	gracefulStop     = make(chan os.Signal, 1)
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
		ScrapeURI:    *scrapeURI,
		HostOverride: *hostOverride,
		Insecure:     *insecure,
	}

	flagConfig := &web.FlagConfig{
		WebConfigFile: configFile,
	}
	exporter := collector.NewExporter(logger, config)
	prometheus.MustRegister(exporter)
	prometheus.MustRegister(version.NewCollector("apache_exporter"))

	level.Info(logger).Log("msg", "Starting apache_exporter", "version", version.Info())
	level.Info(logger).Log("msg", "Build context", "build", version.BuildContext())
	level.Info(logger).Log("msg", "Starting Server: ", "listen_address", *listeningAddress)
	level.Info(logger).Log("msg", "Collect from: ", "scrape_uri", *scrapeURI)

	// listener for the termination signals from the OS
	go func() {
		level.Info(logger).Log("msg", "listening and wait for graceful stop")
		sig := <-gracefulStop
		level.Info(logger).Log("msg", "caught sig: %+v. Wait 2 seconds...", "sig", sig)
		time.Sleep(2 * time.Second)
		level.Info(logger).Log("msg", "Terminate apache-exporter on port:", "listen_address", *listeningAddress)
		os.Exit(0)
	}()

	http.Handle(*metricsEndpoint, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html>
			 <head><title>Apache Exporter</title></head>
			 <body>
			 <h1>Apache Exporter</h1>
			 <p><a href='` + *metricsEndpoint + `'>Metrics</a></p>
			 </body>
			 </html>`))
	})
	//log.Fatal(http.ListenAndServe(*listeningAddress, nil))

	server := &http.Server{Addr: *listeningAddress}

	if err := web.ListenAndServe(server, flagConfig, logger); err != nil {
		level.Error(logger).Log("msg", "Listening error", "reason", err)
		os.Exit(1)
	}

}
