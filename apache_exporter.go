// Copyright (c) 2015 neezgee
//
// Licensed under the MIT license: https://opensource.org/licenses/MIT
// Permission is granted to use, copy, modify, and redistribute the work.
// Full license information available in the project LICENSE file.
//

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

	"os/signal"
	"syscall"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	namespace = "apache" // For Prometheus metrics.
)

var (
	listeningAddress = kingpin.Flag("telemetry.address", "Address on which to expose metrics.").Default(":9117").String()
	metricsEndpoint  = kingpin.Flag("telemetry.endpoint", "Path under which to expose metrics.").Default("/metrics").String()
	scrapeURI        = kingpin.Flag("scrape_uri", "URI to apache stub status page.").Default("http://localhost/server-status/?auto").String()
	hostOverride     = kingpin.Flag("host_override", "Override for HTTP Host header; empty string for no override.").Default("").String()
	insecure         = kingpin.Flag("insecure", "Ignore server certificate if using https.").Bool()
	configFile       = kingpin.Flag("web.config", "Path to config yaml file that can enable TLS or authentication.").Default("").String()
	gracefulStop     = make(chan os.Signal)
)

type Exporter struct {
	URI    string
	mutex  sync.Mutex
	client *http.Client

	up                    *prometheus.Desc
	scrapeFailures        prometheus.Counter
	apacheVersion         *prometheus.Desc
	apacheInfo            *prometheus.GaugeVec
	generation            *prometheus.GaugeVec
	load                  *prometheus.GaugeVec
	accessesTotal         *prometheus.Desc
	kBytesTotal           *prometheus.Desc
	durationTotal         *prometheus.Desc
	cpuTotal              *prometheus.Desc
	cpuload               prometheus.Gauge
	uptime                *prometheus.Desc
	workers               *prometheus.GaugeVec
	processes             *prometheus.GaugeVec
	connections           *prometheus.GaugeVec
	scoreboard            *prometheus.GaugeVec
	proxyBalancerStatus   *prometheus.GaugeVec
	proxyBalancerElected  *prometheus.Desc
	proxyBalancerBusy     *prometheus.GaugeVec
	proxyBalancerReqSize  *prometheus.Desc
	proxyBalancerRespSize *prometheus.Desc
	logger                log.Logger
}

func NewExporter(logger log.Logger, uri string) *Exporter {
	return &Exporter{
		URI:    uri,
		logger: logger,
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
		apacheVersion: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "version"),
			"Apache server version",
			nil,
			nil),
		apacheInfo: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "info",
			Help:      "Apache version information",
		},
			[]string{"version", "mpm"},
		),
		generation: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "generation",
			Help:      "Apache restart generation",
		},
			[]string{"type"},
		),
		load: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "load",
			Help:      "Apache server load",
		},
			[]string{"interval"},
		),
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
		cpuTotal: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "cpu_time_ms_total"),
			"Apache CPU time",
			[]string{"type"}, nil,
		),
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
		processes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "processes",
			Help:      "Apache process count",
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
		scoreboard: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "scoreboard",
			Help:      "Apache scoreboard statuses",
		},
			[]string{"state"},
		),
		proxyBalancerStatus: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "proxy_balancer_status",
			Help:      "Apache Proxy Balancer Statuses",
		},
			[]string{"balancer", "worker", "status"},
		),
		proxyBalancerElected: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "proxy_balancer_accesses_total"),
			"Apache Proxy Balancer Request Count",
			[]string{"balancer", "worker"}, nil,
		),
		proxyBalancerBusy: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "proxy_balancer_busy",
			Help:      "Apache Proxy Balancer Active Requests",
		},
			[]string{"balancer", "worker"},
		),
		proxyBalancerReqSize: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "proxy_balancer_request_kbytes_total"),
			"Apache Proxy Balancer Request Count",
			[]string{"balancer", "worker"}, nil,
		),
		proxyBalancerRespSize: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "proxy_balancer_response_kbytes_total"),
			"Apache Proxy Balancer Request Count",
			[]string{"balancer", "worker"}, nil,
		),
		client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: *insecure},
			},
		},
	}
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.up
	e.scrapeFailures.Describe(ch)
	ch <- e.apacheVersion
	e.apacheInfo.Describe(ch)
	e.generation.Describe(ch)
	e.load.Describe(ch)
	ch <- e.accessesTotal
	ch <- e.kBytesTotal
	ch <- e.durationTotal
	ch <- e.cpuTotal
	e.cpuload.Describe(ch)
	ch <- e.uptime
	e.workers.Describe(ch)
	e.processes.Describe(ch)
	e.connections.Describe(ch)
	e.scoreboard.Describe(ch)
	e.proxyBalancerStatus.Describe(ch)
	ch <- e.proxyBalancerElected
	e.proxyBalancerBusy.Describe(ch)
	ch <- e.proxyBalancerReqSize
	ch <- e.proxyBalancerRespSize
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
	version := "UNKNOWN"
	mpm := "UNKNOWN"
	balancerName := "UNKNOWN"
	workerName := "UNKNOWN"
	cpuUser := 0.0
	cpuSystem := 0.0
	cpuFound := false
	e.proxyBalancerStatus.Reset()
	e.proxyBalancerBusy.Reset()

	for _, l := range lines {
		key, v := splitkv(l)
		if err != nil {
			continue
		}

		switch {
		case key == "ServerVersion":
			var tmpstr string
			var vparts []string

			version = v
			tmpstr = strings.Split(v, "/")[1]
			tmpstr = strings.Split(tmpstr, " ")[0]
			vparts = strings.Split(tmpstr, ".")
			tmpstr = vparts[0] + "." + fmt.Sprintf("%02s", vparts[1]) + fmt.Sprintf("%03s", vparts[2])

			val, err := strconv.ParseFloat(tmpstr, 64)
			if err != nil {
				return err
			}

			ch <- prometheus.MustNewConstMetric(e.apacheVersion, prometheus.GaugeValue, val)
		case key == "ServerMPM":
			mpm = v
		case key == "ParentServerConfigGeneration":
			val, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return err
			}

			e.generation.WithLabelValues("config").Set(val)
		case key == "ParentServerMPMGeneration":
			val, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return err
			}

			e.generation.WithLabelValues("mpm").Set(val)
		case key == "Load1":
			val, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return err
			}

			e.load.WithLabelValues("1min").Set(val)
		case key == "Load5":
			val, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return err
			}

			e.load.WithLabelValues("5min").Set(val)
		case key == "Load15":
			val, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return err
			}

			e.load.WithLabelValues("15min").Set(val)
		case key == "Total Accesses":
			val, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return err
			}

			ch <- prometheus.MustNewConstMetric(e.accessesTotal, prometheus.CounterValue, val)
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
		case key == "CPUUser":
			val, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return err
			}

			cpuUser += val
			cpuFound = true
		case key == "CPUChildrenUser":
			val, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return err
			}

			cpuUser += val
			cpuFound = true
		case key == "CPUSystem":
			val, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return err
			}

			cpuSystem += val
			cpuFound = true
		case key == "CPUChildrenSystem":
			val, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return err
			}

			cpuSystem += val
			cpuFound = true
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
		case key == "Processes":
			val, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return err
			}

			e.processes.WithLabelValues("all").Set(val)
		case key == "Stopping":
			val, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return err
			}

			e.processes.WithLabelValues("stopping").Set(val)
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
		case key == "Scoreboard":
			e.updateScoreboard(v)
			e.scoreboard.Collect(ch)
		//ProxyBalancer[0]Name: balancer://sid2021
		//ProxyBalancer[0]Worker[0]Name: https://z-app-01:9143
		//ProxyBalancer[0]Worker[0]Status: Init Ok
		//ProxyBalancer[0]Worker[0]Elected: 5808
		//...
		case strings.HasPrefix(key, "ProxyBalancer["):
			switch {
			case strings.HasSuffix(key, "]Name"):
				if strings.Contains(key, "]Worker[") {
					workerName = v
				} else {
					balancerName = v
				}
			case strings.HasSuffix(key, "]Status"):
				e.proxyBalancerStatus.WithLabelValues(balancerName, workerName, v).Set(1)
			case strings.HasSuffix(key, "]Elected"):
				val, err := strconv.ParseFloat(v, 64)
				if err != nil {
					return err
				}
				ch <- prometheus.MustNewConstMetric(e.proxyBalancerElected, prometheus.CounterValue, val, balancerName, workerName)
			case strings.HasSuffix(key, "]Busy"):
				val, err := strconv.ParseFloat(v, 64)
				if err != nil {
					return err
				}
				e.proxyBalancerBusy.WithLabelValues(balancerName, workerName).Set(val)
			case strings.HasSuffix(key, "]Sent"):
				val, err := strconv.ParseFloat(strings.TrimRight(v, "kK"), 64)
				if err != nil {
					return err
				}
				ch <- prometheus.MustNewConstMetric(e.proxyBalancerReqSize, prometheus.CounterValue, val, balancerName, workerName)
			case strings.HasSuffix(key, "]Rcvd"):
				val, err := strconv.ParseFloat(strings.TrimRight(v, "kK"), 64)
				if err != nil {
					return err
				}
				ch <- prometheus.MustNewConstMetric(e.proxyBalancerRespSize, prometheus.CounterValue, val, balancerName, workerName)
			}
		}
	}

	if cpuFound {
		ch <- prometheus.MustNewConstMetric(e.cpuTotal, prometheus.CounterValue, 1000*cpuUser, "user")
		ch <- prometheus.MustNewConstMetric(e.cpuTotal, prometheus.CounterValue, 1000*cpuSystem, "system")
	}

	e.apacheInfo.WithLabelValues(version, mpm).Set(1)

	e.apacheInfo.Collect(ch)
	e.generation.Collect(ch)
	e.load.Collect(ch)
	e.cpuload.Collect(ch)
	e.workers.Collect(ch)
	e.processes.Collect(ch)
	if connectionInfo {
		e.connections.Collect(ch)
	}

	e.proxyBalancerStatus.Collect(ch)
	e.proxyBalancerBusy.Collect(ch)

	return nil
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.mutex.Lock() // To protect metrics from concurrent collects.
	defer e.mutex.Unlock()
	if err := e.collect(ch); err != nil {
		level.Error(e.logger).Log("msg", "Error scraping apache:", "err", err)
		e.scrapeFailures.Inc()
		e.scrapeFailures.Collect(ch)
	}
	return
}

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

	exporter := NewExporter(logger, *scrapeURI)
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

	if err := web.ListenAndServe(server, *configFile, logger); err != nil {
		level.Error(logger).Log("msg", "Listening error", "reason", err)
		os.Exit(1)
	}

}
