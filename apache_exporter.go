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
	listeningAddress = flag.String("telemetry.address", ":9117", "Address on which to expose metrics.")
	metricsEndpoint  = flag.String("telemetry.endpoint", "/metrics", "Path under which to expose metrics.")
	scrapeURI        = flag.String("scrape_uri", "http://localhost/server-status/?auto", "URI to apache stub status page.")
	insecure         = flag.Bool("insecure", false, "Ignore server certificate if using https.")
)

type Exporter struct {
	URI    string
	mutex  sync.Mutex
	client *http.Client

	scrapeFailures prometheus.Counter
	accessesTotal  prometheus.Counter
	kBytesTotal    prometheus.Counter
	uptime         prometheus.Counter
	workers        *prometheus.GaugeVec
	scoreboard     *prometheus.GaugeVec
	connections    *prometheus.GaugeVec
}

func NewExporter(uri string) *Exporter {
	return &Exporter{
		URI: uri,
		scrapeFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "exporter_scrape_failures_total",
			Help:      "Number of errors while scraping apache.",
		}),
		accessesTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "accesses_total",
			Help:      "Current total apache accesses",
		}),
		kBytesTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "sent_kilobytes_total",
			Help:      "Current total kbytes sent",
		}),
		uptime: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "uptime_seconds_total",
			Help:      "Current uptime in seconds",
		}),
		workers: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "workers",
			Help:      "Apache worker statuses",
		},
			[]string{"state"},
		),
		scoreboard: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "scoreboard",
			Help:      "Apache scoreboard statuses",
		},
			[]string{"state"},
		),
		connections: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "connections",
			Help:      "Apache connection statuses",
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

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	e.scrapeFailures.Describe(ch)
	e.accessesTotal.Describe(ch)
	e.kBytesTotal.Describe(ch)
	e.uptime.Describe(ch)
	e.workers.Describe(ch)
	e.scoreboard.Describe(ch)
	e.connections.Describe(ch)
}

// Split colon separated string into two fields
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

func (e *Exporter) updateScoreboard(scoreboard string) {
	e.scoreboard.Reset()
	for _, worker_status := range scoreboard {
		s := string(worker_status)
		switch {
		case s == "_":
			e.scoreboard.WithLabelValues("idle").Inc()
		case s == "S":
			e.scoreboard.WithLabelValues("startup").Inc()
		case s == "R":
			e.scoreboard.WithLabelValues("read").Inc()
		case s == "W":
			e.scoreboard.WithLabelValues("reply").Inc()
		case s == "K":
			e.scoreboard.WithLabelValues("keepalive").Inc()
		case s == "D":
			e.scoreboard.WithLabelValues("dns").Inc()
		case s == "C":
			e.scoreboard.WithLabelValues("closing").Inc()
		case s == "L":
			e.scoreboard.WithLabelValues("logging").Inc()
		case s == "G":
			e.scoreboard.WithLabelValues("graceful_stop").Inc()
		case s == "I":
			e.scoreboard.WithLabelValues("idle_cleanup").Inc()
		case s == ".":
			e.scoreboard.WithLabelValues("open_slot").Inc()
		}
	}
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

        connectionInfo := false

	for _, l := range lines {
		key, v := splitkv(l)
		if err != nil {
			continue
		}

		switch {
		case key == "Total Accesses":
			val, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return err
			}

			e.accessesTotal.Set(val)
			e.accessesTotal.Collect(ch)
		case key == "Total kBytes":
			val, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return err
			}

			e.kBytesTotal.Set(val)
			e.kBytesTotal.Collect(ch)
		case key == "Uptime":
			val, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return err
			}

			e.uptime.Set(val)
			e.uptime.Collect(ch)
		case key == "BusyWorkers":
			val, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return err
			}

			e.workers.WithLabelValues("busy").Set(val)
		case key == "IdleWorkers":
			val, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return err
			}

			e.workers.WithLabelValues("idle").Set(val)
		case key == "Scoreboard":
			e.updateScoreboard(v)
			e.scoreboard.Collect(ch)
		case key == "ConnsTotal":
			val, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return err
			}

			e.connections.WithLabelValues("total").Set(val)
                        connectionInfo = true
		case key == "ConnsAsyncWriting":
			val, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return err
			}

			e.connections.WithLabelValues("writing").Set(val)
                        connectionInfo = true
		case key == "ConnsAsyncKeepAlive":
			val, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return err
			}
			e.connections.WithLabelValues("keepalive").Set(val)
                        connectionInfo = true
		case key == "ConnsAsyncClosing":
			val, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return err
			}
			e.connections.WithLabelValues("closing").Set(val)
                        connectionInfo = true
		}


	}

	e.workers.Collect(ch)
        if connectionInfo {
		e.connections.Collect(ch)
        }

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
