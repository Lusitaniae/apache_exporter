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

	metricCountApache22 = 5
        metricCountApache24 = 9

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
