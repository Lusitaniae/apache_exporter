package main

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
	"gopkg.in/alecthomas/kingpin.v2"
	"os/signal"
	"syscall"
	"time"
)

const (
	namespace = "apache" // For Prometheus metrics.
)

func getDefaultFromEnv(environment, defaultValue string) string {
	value, found := os.LookupEnv(environment)
	if found {
		return value
	}
	return defaultValue
}

var (
	listeningAddress = kingpin.Flag("telemetry.address", "Address on which to expose metrics.").Default(getDefaultFromEnv("TELEMETRY_ADDRESS", ":9117")).String()
	metricsEndpoint  = kingpin.Flag("telemetry.endpoint", "Path under which to expose metrics.").Default(getDefaultFromEnv("TELEMETRY_ENDPOINT", "/metrics")).String()
	scrapeURI        = kingpin.Flag("scrape_uri", "URI to apache stub status page.").Default(getDefaultFromEnv("SCRAPE_URI", "http://localhost/server-status/?auto")).String()
	hostOverride     = kingpin.Flag("host_override", "Override for HTTP Host header; empty string for no override.").Default(getDefaultFromEnv("HOST_OVERRIDE", "")).String()
	insecure         = kingpin.Flag("insecure", "Ignore server certificate if using https.").Default(getDefaultFromEnv("INSECURE", "false")).Bool()
	gracefulStop     = make(chan os.Signal)
)

type Exporter struct {
	URI    string
	mutex  sync.Mutex
	client *http.Client

	up             *prometheus.Desc
	scrapeFailures prometheus.Counter
	accessesTotal  *prometheus.Desc
	kBytesTotal    *prometheus.Desc
	durationTotal  *prometheus.Desc
	cpuload        prometheus.Gauge
	uptime         *prometheus.Desc
	workers        *prometheus.GaugeVec
	scoreboard     *prometheus.GaugeVec
	connections    *prometheus.GaugeVec

	apacheVersion *prometheus.Desc
}

func NewExporter(uri string) *Exporter {
	return &Exporter{
		URI: uri,
		up: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "up"),
			"Could the apache server be reached",
			nil,
			nil),
		scrapeFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "exporter_scrape_failures_total",
			Help:      "Number of errors while scraping apache.",
		}),
		accessesTotal: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "accesses_total"),
			"Current total apache accesses (*)",
			nil,
			nil),
		kBytesTotal: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "sent_kilobytes_total"),
			"Current total kbytes sent (*)",
			nil,
			nil),
		durationTotal: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "duration_ms_total"),
			"Total duration of all registered requests in ms",
			nil,
			nil),
		cpuload: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "cpuload",
			Help:      "The current percentage CPU used by each worker and in total by all workers combined (*)",
		}),
		uptime: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "uptime_seconds_total"),
			"Current uptime in seconds (*)",
			nil,
			nil),
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
		apacheVersion: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "version"),
			"Apache server version",
			[]string{"version"},
			nil),
		client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: *insecure},
			},
		},
	}
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.up
	ch <- e.accessesTotal
	ch <- e.kBytesTotal
	ch <- e.uptime
	ch <- e.durationTotal
	ch <- e.apacheVersion
	e.cpuload.Describe(ch)
	e.scrapeFailures.Describe(ch)
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

var scoreboardLabelMap = map[string]string{
	"_": "idle",
	"S": "startup",
	"R": "read",
	"W": "reply",
	"K": "keepalive",
	"D": "dns",
	"C": "closing",
	"L": "logging",
	"G": "graceful_stop",
	"I": "idle_cleanup",
	".": "open_slot",
}

func (e *Exporter) updateScoreboard(scoreboard string) {
	e.scoreboard.Reset()
	for _, v := range scoreboardLabelMap {
		e.scoreboard.WithLabelValues(v)
	}

	for _, worker_status := range scoreboard {
		s := string(worker_status)
		label, ok := scoreboardLabelMap[s]
		if !ok {
			label = s
		}
		e.scoreboard.WithLabelValues(label).Inc()
	}
}

func (e *Exporter) collect(ch chan<- prometheus.Metric) error {
	req, err := http.NewRequest("GET", e.URI, nil)
	if *hostOverride != "" {
		req.Host = *hostOverride
	}
	if err != nil {
		return fmt.Errorf("error building scraping request: %v", err)
	}
	resp, err := e.client.Do(req)
	if err != nil {
		ch <- prometheus.MustNewConstMetric(e.up, prometheus.GaugeValue, 0)
		return fmt.Errorf("error scraping apache: %v", err)
	}
	ch <- prometheus.MustNewConstMetric(e.up, prometheus.GaugeValue, 1)

	data, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 {
		if err != nil {
			data = []byte(err.Error())
		}
		return fmt.Errorf("status %s (%d): %s", resp.Status, resp.StatusCode, data)
	}

	lines := strings.Split(string(data), "\n")

	connectionInfo := false
	//connectionInfo := false

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

			ch <- prometheus.MustNewConstMetric(e.accessesTotal, prometheus.CounterValue, val)
		case key == "ServerVersion":
			var minVersion string

			minVersion = strings.Split(v, "/")[1]
			minVersion = strings.Split(minVersion, " ")[0]
			minVersion = strings.Join(strings.Split(minVersion, ".")[0:2], ".")

			val, err := strconv.ParseFloat(minVersion, 64)
			if err != nil {
				return err
			}

			ch <- prometheus.MustNewConstMetric(e.apacheVersion, prometheus.CounterValue, val, v)
		case key == "Total kBytes":
			val, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return err
			}

			ch <- prometheus.MustNewConstMetric(e.kBytesTotal, prometheus.CounterValue, val)
		case key == "Total Duration":
			val, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return err
			}

			ch <- prometheus.MustNewConstMetric(e.durationTotal, prometheus.CounterValue, val)
		case key == "CPULoad":
			val, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return err
			}

			e.cpuload.Set(val)
		case key == "Uptime":
			val, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return err
			}

			ch <- prometheus.MustNewConstMetric(e.uptime, prometheus.CounterValue, val)
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

	e.cpuload.Collect(ch)
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
		log.Errorf("Error scraping apache: %s", err)
		e.scrapeFailures.Inc()
		e.scrapeFailures.Collect(ch)
	}
	return
}

func main() {

	// Parse flags
	log.AddFlags(kingpin.CommandLine)
	kingpin.Version(version.Print("apache_exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	// listen to termination signals from the OS
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)
	signal.Notify(gracefulStop, syscall.SIGHUP)
	signal.Notify(gracefulStop, syscall.SIGQUIT)

	exporter := NewExporter(*scrapeURI)
	prometheus.MustRegister(exporter)
	prometheus.MustRegister(version.NewCollector("apache_exporter"))

	log.Infoln("Starting apache_exporter", version.Info())
	log.Infoln("Build context", version.BuildContext())
	log.Infof("Starting Server: %s", *listeningAddress)
	log.Infof("Collect from: %s", *scrapeURI)

	// listener for the termination signals from the OS
	go func() {
		log.Infof("listening and wait for graceful stop")
		sig := <-gracefulStop
		log.Infof("caught sig: %+v. Wait 2 seconds...", sig)
		time.Sleep(2 * time.Second)
		log.Infof("Terminate apache-exporter on port: %s", *listeningAddress)
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
	log.Fatal(http.ListenAndServe(*listeningAddress, nil))
}
