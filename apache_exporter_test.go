// Copyright (c) 2015 neezgee
//
// Licensed under the MIT license: https://opensource.org/licenses/MIT
// Permission is granted to use, copy, modify, and redistribute the work.
// Full license information available in the project LICENSE file.
//

package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Lusitaniae/apache_exporter/collector"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/promlog"
)

const (
	apache24EventStatus = `localhost
ServerVersion: Apache/2.4.23 (Unix)
ServerMPM: event
Server Built: Jul 29 2016 04:26:14
CurrentTime: Friday, 29-Jul-2016 14:06:15 UTC
RestartTime: Friday, 29-Jul-2016 13:58:49 UTC
ParentServerConfigGeneration: 1
ParentServerMPMGeneration: 0
ServerUptimeSeconds: 445
ServerUptime: 7 minutes 25 seconds
Load1: 0.02
Load5: 0.02
Load15: 0.00
Total Accesses: 131
Total kBytes: 138
Total Duration: 12930
CPUUser: .25
CPUSystem: .15
CPUChildrenUser: 0
CPUChildrenSystem: 0
CPULoad: .0898876
Uptime: 445
ReqPerSec: .294382
BytesPerSec: 317.555
BytesPerReq: 1078.72
DurationPerReq: 98.7022
BusyWorkers: 1
IdleWorkers: 74
Processes: 5
Stopping: 0
ConnsTotal: 0
ConnsAsyncWriting: 0
ConnsAsyncKeepAlive: 0
ConnsAsyncClosing: 0
Scoreboard: _W___
`

	apache24EventTLSStatus = `localhost
ServerVersion: Apache/2.4.23 (Unix)
ServerMPM: event
Server Built: Jul 29 2016 04:26:14
CurrentTime: Friday, 29-Jul-2016 14:06:15 UTC
RestartTime: Friday, 29-Jul-2016 13:58:49 UTC
ParentServerConfigGeneration: 1
ParentServerMPMGeneration: 0
ServerUptimeSeconds: 445
ServerUptime: 7 minutes 25 seconds
Load1: 0.02
Load5: 0.02
Load15: 0.00
Total Accesses: 131
Total kBytes: 138
Total Duration: 12930
CPUUser: .25
CPUSystem: .15
CPUChildrenUser: 0
CPUChildrenSystem: 0
CPULoad: .0898876
Uptime: 445
ReqPerSec: .294382
BytesPerSec: 317.555
BytesPerReq: 1078.72
DurationPerReq: 98.7022
BusyWorkers: 1
IdleWorkers: 74
Processes: 5
Stopping: 0
ConnsTotal: 0
ConnsAsyncWriting: 0
ConnsAsyncKeepAlive: 0
ConnsAsyncClosing: 0
Scoreboard: _W___
TLSSessionCacheStatus
CacheType: SHMCB
CacheSharedMemory: 512000
CacheCurrentEntries: 0
CacheSubcaches: 32
CacheIndexesPerSubcaches: 88
CacheIndexUsage: 0%
CacheUsage: 0%
CacheStoreCount: 0
CacheReplaceCount: 0
CacheExpireCount: 0
CacheDiscardCount: 0
CacheRetrieveHitCount: 0
CacheRetrieveMissCount: 1
CacheRemoveHitCount: 0
CacheRemoveMissCount: 0
`

	apache24EventProxyStatus = `localhost
ServerVersion: Apache/2.4.23 (Unix)
ServerMPM: event
Server Built: Jul 29 2016 04:26:14
CurrentTime: Friday, 29-Jul-2016 14:06:15 UTC
RestartTime: Friday, 29-Jul-2016 13:58:49 UTC
ParentServerConfigGeneration: 1
ParentServerMPMGeneration: 0
ServerUptimeSeconds: 445
ServerUptime: 7 minutes 25 seconds
Load1: 0.02
Load5: 0.02
Load15: 0.00
Total Accesses: 131
Total kBytes: 138
Total Duration: 12930
CPUUser: .25
CPUSystem: .15
CPUChildrenUser: 0
CPUChildrenSystem: 0
CPULoad: .0898876
Uptime: 445
ReqPerSec: .294382
BytesPerSec: 317.555
BytesPerReq: 1078.72
DurationPerReq: 98.7022
BusyWorkers: 1
IdleWorkers: 74
Processes: 5
Stopping: 0
ConnsTotal: 0
ConnsAsyncWriting: 0
ConnsAsyncKeepAlive: 0
ConnsAsyncClosing: 0
Scoreboard: _W___
ProxyBalancer[0]Name: balancer://myproxy1
ProxyBalancer[0]Worker[0]Name: https://app-01:9143
ProxyBalancer[0]Worker[0]Status: Init Ok
ProxyBalancer[0]Worker[0]Elected: 5808
ProxyBalancer[0]Worker[0]Busy: 0
ProxyBalancer[0]Worker[0]Sent: 5588K
ProxyBalancer[0]Worker[0]Rcvd: 8335K
ProxyBalancer[0]Worker[1]Name: https://app-02:9143
ProxyBalancer[0]Worker[1]Status: Init Ok
ProxyBalancer[0]Worker[1]Elected: 5722
ProxyBalancer[0]Worker[1]Busy: 0
ProxyBalancer[0]Worker[1]Sent: 5167K
ProxyBalancer[0]Worker[1]Rcvd: 8267K
ProxyBalancer[0]Worker[2]Name: https://app-03:9143
ProxyBalancer[0]Worker[2]Status: Init Ok
ProxyBalancer[0]Worker[2]Elected: 5842
ProxyBalancer[0]Worker[2]Busy: 0
ProxyBalancer[0]Worker[2]Sent: 5432K
ProxyBalancer[0]Worker[2]Rcvd: 8367K
ProxyBalancer[0]Worker[3]Name: https://app-04:9143
ProxyBalancer[0]Worker[3]Status: Init Ok
ProxyBalancer[0]Worker[3]Elected: 5720
ProxyBalancer[0]Worker[3]Busy: 0
ProxyBalancer[0]Worker[3]Sent: 5576K
ProxyBalancer[0]Worker[3]Rcvd: 8175K
ProxyBalancer[1]Name: balancer://myproxy2
ProxyBalancer[1]Worker[0]Name: https://app-01:8143
ProxyBalancer[1]Worker[0]Status: Init Ok
ProxyBalancer[1]Worker[0]Elected: 5808
ProxyBalancer[1]Worker[0]Busy: 0
ProxyBalancer[1]Worker[0]Sent: 5588K
ProxyBalancer[1]Worker[0]Rcvd: 8335K
ProxyBalancer[1]Worker[1]Name: https://app-02:8143
ProxyBalancer[1]Worker[1]Status: Init Ok
ProxyBalancer[1]Worker[1]Elected: 5722
ProxyBalancer[1]Worker[1]Busy: 0
ProxyBalancer[1]Worker[1]Sent: 5167K
ProxyBalancer[1]Worker[1]Rcvd: 8267K
`

	apache24WorkerStatus = `localhost
ServerVersion: Apache/2.4.23 (Unix) OpenSSL/1.0.2h
ServerMPM: worker
Server Built: Aug 31 2016 10:54:08
CurrentTime: Thursday, 08-Sep-2016 15:09:32 CEST
RestartTime: Thursday, 08-Sep-2016 15:08:07 CEST
ParentServerConfigGeneration: 1
ParentServerMPMGeneration: 0
ServerUptimeSeconds: 85
ServerUptime: 1 minute 25 seconds
Load1: 0.00
Load5: 0.01
Load15: 0.05
Total Accesses: 10
Total kBytes: 38
Total Duration: 12930
CPUUser: .05
CPUSystem: 0
CPUChildrenUser: 0
CPUChildrenSystem: 0
CPULoad: .0588235
Uptime: 85
ReqPerSec: .117647
BytesPerSec: 457.788
BytesPerReq: 3891.2
DurationPerReq: 1293.00
BusyWorkers: 2
IdleWorkers: 48
Scoreboard: _____R_______________________K____________________....................................................................................................
`

	apache24PreforkStatus = `localhost
ServerVersion: Apache/2.4.23 (Unix) OpenSSL/1.0.2h
ServerMPM: prefork
Server Built: Aug 31 2016 10:54:08
CurrentTime: Thursday, 08-Sep-2016 15:09:32 CEST
RestartTime: Thursday, 08-Sep-2016 15:08:07 CEST
ParentServerConfigGeneration: 1
ParentServerMPMGeneration: 0
ServerUptimeSeconds: 85
ServerUptime: 1 minute 25 seconds
Load1: 0.00
Load5: 0.01
Load15: 0.05
Total Accesses: 10
Total kBytes: 38
Total Duration: 12930
CPUUser: .05
CPUSystem: 0
CPUChildrenUser: 0
CPUChildrenSystem: 0
CPULoad: .0588235
Uptime: 85
ReqPerSec: .117647
BytesPerSec: 457.788
BytesPerReq: 3891.2
DurationPerReq: 1293.00
BusyWorkers: 2
IdleWorkers: 48
Scoreboard: _____R_______________________K____________________....................................................................................................
`

	apache22Status = `Total Accesses: 302311
Total kBytes: 1677830
CPULoad: 27.4052
Uptime: 45683
ReqPerSec: 6.61758
BytesPerSec: 37609.1
BytesPerReq: 5683.21
BusyWorkers: 2
IdleWorkers: 8
Scoreboard: _W_______K......................................................................................................................................................................................................................................................
`

	metricCountApache22           = 19
	metricCountApache24Event      = 34
	metricCountApache24EventTLS   = 34
	metricCountApache24EventProxy = 64
	metricCountApache24Worker     = 28
	metricCountApache24Prefork    = 28
)

func checkApacheStatus(t *testing.T, status string, metricCount int) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(status))
	})
	server := httptest.NewServer(handler)
	promlogConfig := &promlog.Config{}
	logger := promlog.New(promlogConfig)
	config := &collector.Config{
		ScrapeURI:     server.URL,
		HostOverride:  "",
		Insecure:      false,
		CustomHeaders: map[string]string{"Cookie": "A test cookie"},
	}
	e := collector.NewExporter(logger, config)
	ch := make(chan prometheus.Metric)

	go func() {
		defer close(ch)
		e.Collect(ch)
	}()

	for i := 1; i <= metricCount; i++ {
		m := <-ch
		if m == nil {
			t.Error("expected metric but got nil")
		}
	}
	extraMetrics := 0
	for <-ch != nil {
		extraMetrics++
	}
	if extraMetrics > 0 {
		t.Errorf("expected closed channel, got %d extra metrics", extraMetrics)
	}
}

func TestApache22Status(t *testing.T) {
	checkApacheStatus(t, apache22Status, metricCountApache22)
}

func TestApache24EventStatus(t *testing.T) {
	checkApacheStatus(t, apache24EventStatus, metricCountApache24Event)
}

func TestApache24EventTLSStatus(t *testing.T) {
	checkApacheStatus(t, apache24EventTLSStatus, metricCountApache24EventTLS)
}

func TestApache24EventProxyStatus(t *testing.T) {
	checkApacheStatus(t, apache24EventProxyStatus, metricCountApache24EventProxy)
}

func TestApache24WorkerStatus(t *testing.T) {
	checkApacheStatus(t, apache24WorkerStatus, metricCountApache24Worker)
}

func TestApache24PreforkStatus(t *testing.T) {
	checkApacheStatus(t, apache24PreforkStatus, metricCountApache24Prefork)
}
