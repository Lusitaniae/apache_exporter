package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	apache24Status = `localhost
ServerVersion: Apache/2.4.16 (Unix)
ServerMPM: prefork
Server Built: Jul 22 2015 21:03:09
CurrentTime: Monday, 16-May-2016 18:37:02 JST
RestartTime: Monday, 16-May-2016 16:36:41 JST
ParentServerConfigGeneration: 1
ParentServerMPMGeneration: 0
ServerUptimeSeconds: 7220
ServerUptime: 2 hours 20 seconds
Load1: 3.23
Load5: 3.29
Load15: 2.89
Total Accesses: 1
Total kBytes: 2
CPUUser: 0
CPUSystem: .03
CPUChildrenUser: 0
CPUChildrenSystem: 0
CPULoad: .000415512
Uptime: 15664
ReqPerSec: 6.38407e-5
BytesPerSec: .130746
BytesPerReq: 2048
BusyWorkers: 1
IdleWorkers: 4
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

	metricCount = 5
)

func checkApacheStatus(t *testing.T, status string) {
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
	checkApacheStatus(t, apache22Status)
}

func TestApache24Status(t *testing.T) {
	checkApacheStatus(t, apache24Status)
}
