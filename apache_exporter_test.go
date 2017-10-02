package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	apache24Status = `localhost
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
CPUUser: .25
CPUSystem: .15
CPUChildrenUser: 0
CPUChildrenSystem: 0
CPULoad: .0898876
Uptime: 445
ReqPerSec: .294382
BytesPerSec: 317.555
BytesPerReq: 1078.72
BusyWorkers: 1
IdleWorkers: 74
ConnsTotal: 0
ConnsAsyncWriting: 0
ConnsAsyncKeepAlive: 0
ConnsAsyncClosing: 0
Scoreboard: _W___
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
CPUUser: .05
CPUSystem: 0
CPUChildrenUser: 0
CPUChildrenSystem: 0
CPULoad: .0588235
Uptime: 85
ReqPerSec: .117647
BytesPerSec: 457.788
BytesPerReq: 3891.2
BusyWorkers: 2
IdleWorkers: 48
Scoreboard: _____R_______________________K____________________....................................................................................................
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

	metricCountApache22       = 10
	metricCountApache24       = 12
	metricCountApache24Worker = 10
)

func checkApacheStatus(t *testing.T, status string, metricCount int) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(status))
	})
	server := httptest.NewServer(handler)

	e := NewExporter(server.URL)
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
	if <-ch != nil {
		t.Error("expected closed channel")
	}
}

func TestApache22Status(t *testing.T) {
	checkApacheStatus(t, apache22Status, metricCountApache22)
}

func TestApache24Status(t *testing.T) {
	checkApacheStatus(t, apache24Status, metricCountApache24)
}

func TestApache24WorkerStatus(t *testing.T) {
	checkApacheStatus(t, apache24WorkerStatus, metricCountApache24Worker)
}
