// Copyright (c) 2015 neezgee
//
// Licensed under the MIT license: https://opensource.org/licenses/MIT
// Permission is granted to use, copy, modify, and redistribute the work.
// Full license information available in the project LICENSE file.
//

package collector

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/prometheus/common/promslog"
)

// TestApacheInfoAfterUpgrade checks that apache_info only reports the current
// version once Apache has been upgraded, instead of keeping the old series
// (https://github.com/Lusitaniae/apache_exporter/issues/115).
func TestApacheInfoAfterUpgrade(t *testing.T) {
	var status atomic.Value
	status.Store("ServerVersion: Apache/2.4.50 (Debian)\nServerMPM: worker\n")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(status.Load().(string)))
	}))
	defer server.Close()

	e := NewExporter(promslog.NewNopLogger(), &Config{ScrapeURI: server.URL})

	const before = `
# HELP apache_info Apache version information
# TYPE apache_info gauge
apache_info{mpm="worker",version="Apache/2.4.50 (Debian)"} 1
`
	if err := testutil.CollectAndCompare(e, strings.NewReader(before), "apache_info"); err != nil {
		t.Fatal(err)
	}

	status.Store("ServerVersion: Apache/2.4.51 (Debian)\nServerMPM: event\n")

	const after = `
# HELP apache_info Apache version information
# TYPE apache_info gauge
apache_info{mpm="event",version="Apache/2.4.51 (Debian)"} 1
`
	if err := testutil.CollectAndCompare(e, strings.NewReader(after), "apache_info"); err != nil {
		t.Fatal(err)
	}
}
