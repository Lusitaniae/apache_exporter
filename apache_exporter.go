package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/log"
)

const (
	namespace = "apache" // For Prometheus metrics.
)

var (
	listeningAddress = flag.String("telemetry.address", ":9114", "Address on which to expose metrics.")
	metricsEndpoint  = flag.String("telemetry.endpoint", "/metrics", "Path under which to expose metrics.")
	scrapeURI        = flag.String("scrape_uri", "http://localhost/server-status/?auto", "URI to apache stub status page")
	insecure         = flag.Bool("insecure", true, "Ignore server certificate if using https")
)

// Exporter collects apache stats from the given URI and exports them using
// the prometheus metrics package.

type Exporter struct {
	URI    string
	mutex  sync.RWMutex
	client *http.Client

	scrapeFailures prometheus.Counter
	totalAccesses  prometheus.Counter
	totalKBytes    prometheus.Counter
	uptime         prometheus.Counter
	reqPerSec      prometheus.Gauge
	bytesPerSec    prometheus.Gauge
	bytesPerReq    prometheus.Gauge
	workers        *prometheus.GaugeVec
}

/*
Total Accesses: 1
Total kBytes: 2
Uptime: 15664
ReqPerSec: 6.38407e-5
BytesPerSec: .130746
BytesPerReq: 2048
BusyWorkers: 1
IdleWorkers: 4
Scoreboard: _W___

Total Accesses: 302311
Total kBytes: 1677830
CPULoad: 27.4052
Uptime: 45683
ReqPerSec: 6.61758
BytesPerSec: 37609.1
BytesPerReq: 5683.21
BusyWorkers: 2
IdleWorkers: 8
Scoreboard: _W_______K......................................................................................................................................................................................................................................................

*/

// NewExporter returns an initialized Exporter.
func NewExporter(uri string) *Exporter {
	return &Exporter{
		URI: uri,
		scrapeFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "exporter_scrape_failures_total",
			Help:      "Number of errors while scraping apache.",
		}),
		totalAccesses: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "access_total",
			Help:      "Current total apache access",
		}),
		totalKBytes: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "sent_total_kilobytes",
			Help:      "Current total kbytes sent",
		}),
		uptime: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "uptime_seconds",
			Help:      "Current uptime in seconds",
		}),
		reqPerSec: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "requests_per_second",
			Help:      "Requests per second",
		}),
		bytesPerSec: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "sent_per_second_bytes",
			Help:      "Requests per second",
		}),
		bytesPerReq: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "request_size_bytes",
			Help:      "Bytes per request",
		}),
		workers: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "workers",
			Help:      "Apache worker statuses",
		},
			[]string{"state"},
		),
		client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: *insecure},
			},
		},
	}
}

// Describe describes all the metrics ever exported by the apache exporter. It
// implements prometheus.Collector.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	e.scrapeFailures.Describe(ch)
	e.totalAccesses.Describe(ch)
	e.totalKBytes.Describe(ch)
	e.uptime.Describe(ch)
	e.reqPerSec.Describe(ch)
	e.bytesPerSec.Describe(ch)
	e.bytesPerReq.Describe(ch)
	e.workers.Describe(ch)
}

func splitkv(s string) (string, string) {

	if len(s) == 0 {
		return s, s
	}

	slice := strings.SplitN(s, ":", 2)

	if len(slice) == 1 {
		return slice[0], ""
	}

	return strings.TrimSpace(slice[0]), strings.TrimSpace(slice[1])
}

func (e *Exporter) collect(ch chan<- prometheus.Metric) error {
	resp, err := e.client.Get(e.URI)
	if err != nil {
		return fmt.Errorf("Error scraping apache: %v", err)
	}

	data, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 {
		if err != nil {
			data = []byte(err.Error())
		}
		return fmt.Errorf("Status %s (%d): %s", resp.Status, resp.StatusCode, data)
	}

	lines := strings.Split(string(data), "\n")

	for _, l := range lines {
		key, v := splitkv(l)
		if key == "Scoreboard" || key == "" {
			continue
		}

		val, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return err
		}

		switch {
		case key == "Total Accesses":
			e.totalAccesses.Set(val)
			e.totalAccesses.Collect(ch)
		case key == "Total kBytes":
			e.totalKBytes.Set(val)
			e.totalKBytes.Collect(ch)
		case key == "Uptime":
			e.uptime.Set(val)
			e.uptime.Collect(ch)
		case key == "ReqPerSec":
			e.reqPerSec.Set(val)
			e.reqPerSec.Collect(ch)
		case key == "BytesPerSec":
			e.bytesPerSec.Set(val)
			e.bytesPerSec.Collect(ch)
		case key == "BytesPerReq":
			e.bytesPerReq.Set(val)
			e.bytesPerReq.Collect(ch)
		case key == "BusyWorkers":
			e.workers.WithLabelValues("busy").Set(val)
		case key == "IdleWorkers":
			e.workers.WithLabelValues("idle").Set(val)
		}
	}

	e.workers.Collect(ch)

	return nil
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.mutex.Lock() // To protect metrics from concurrent collects.
	defer e.mutex.Unlock()
	if err := e.collect(ch); err != nil {
		log.Printf("Error scraping apache: %s", err)
		e.scrapeFailures.Inc()
		e.scrapeFailures.Collect(ch)
	}
	return
}

func main() {
	flag.Parse()

	exporter := NewExporter(*scrapeURI)
	prometheus.MustRegister(exporter)

	log.Printf("Starting Server: %s", *listeningAddress)
	http.Handle(*metricsEndpoint, prometheus.Handler())
	log.Fatal(http.ListenAndServe(*listeningAddress, nil))
}
