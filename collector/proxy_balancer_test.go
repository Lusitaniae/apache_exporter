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
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/prometheus/common/promslog"
)

func TestParseProxyKBytes(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want float64
	}{
		// httpd >= 2.4.35: whole kilobytes.
		{"5588K", 5588},
		{"3105327K", 3105327},
		{"0K", 0},
		// httpd < 2.4.35: apr_strfsize output.
		{"  0 ", 0},
		{"120", 120.0 / 1024},
		{"1.2K", 1.2},
		{" 12K", 12},
		{"3.9M", 3.9 * 1024},
		{"512M", 512 * 1024},
		{"1.1G", 1.1 * 1024 * 1024},
		{"2.0T", 2.0 * 1024 * 1024 * 1024},
		// Lowercase suffix, accepted before this parser existed.
		{"42k", 42},
	} {
		got, err := parseProxyKBytes(tc.in)
		if err != nil {
			t.Errorf("parseProxyKBytes(%q): unexpected error: %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("parseProxyKBytes(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}

	for _, in := range []string{"", "K", "-", "  - ", "****", "1.2X"} {
		if got, err := parseProxyKBytes(in); err == nil {
			t.Errorf("parseProxyKBytes(%q) = %v, want error", in, got)
		}
	}
}

// TestProxyBalancerSizeSuffixes checks that M/G-suffixed and plain-byte
// Sent/Rcvd values from httpd < 2.4.35 are exported instead of failing the
// scrape (https://github.com/Lusitaniae/apache_exporter/issues/192).
func TestProxyBalancerSizeSuffixes(t *testing.T) {
	const status = `ServerVersion: Apache/2.4.25 (Debian)
ServerMPM: event
ProxyBalancer[0]Name: balancer://legacy
ProxyBalancer[0]Worker[0]Name: http://app-01:8080
ProxyBalancer[0]Worker[0]Status: Init Ok
ProxyBalancer[0]Worker[0]Elected: 10
ProxyBalancer[0]Worker[0]Busy: 0
ProxyBalancer[0]Worker[0]Sent: 3.9M
ProxyBalancer[0]Worker[0]Rcvd: 120
ProxyBalancer[0]Worker[1]Name: http://app-02:8080
ProxyBalancer[0]Worker[1]Status: Init Ok
ProxyBalancer[0]Worker[1]Elected: 20
ProxyBalancer[0]Worker[1]Busy: 0
ProxyBalancer[0]Worker[1]Sent:  12K
ProxyBalancer[0]Worker[1]Rcvd: 1.1G
ProxyBalancer[1]Name: balancer://current
ProxyBalancer[1]Worker[0]Name: http://app-03:8080
ProxyBalancer[1]Worker[0]Status: Init Ok
ProxyBalancer[1]Worker[0]Elected: 30
ProxyBalancer[1]Worker[0]Busy: 0
ProxyBalancer[1]Worker[0]Sent: 5588K
ProxyBalancer[1]Worker[0]Rcvd: 3105327K
`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(status))
	}))
	defer server.Close()

	e := NewExporter(promslog.NewNopLogger(), &Config{ScrapeURI: server.URL})

	const want = `
# HELP apache_proxy_balancer_request_kbytes_total Apache Proxy Balancer Request Count
# TYPE apache_proxy_balancer_request_kbytes_total counter
apache_proxy_balancer_request_kbytes_total{balancer="balancer://legacy",worker="http://app-01:8080"} 3993.6
apache_proxy_balancer_request_kbytes_total{balancer="balancer://legacy",worker="http://app-02:8080"} 12
apache_proxy_balancer_request_kbytes_total{balancer="balancer://current",worker="http://app-03:8080"} 5588
# HELP apache_proxy_balancer_response_kbytes_total Apache Proxy Balancer Request Count
# TYPE apache_proxy_balancer_response_kbytes_total counter
apache_proxy_balancer_response_kbytes_total{balancer="balancer://legacy",worker="http://app-01:8080"} 0.1171875
apache_proxy_balancer_response_kbytes_total{balancer="balancer://legacy",worker="http://app-02:8080"} 1153433.6
apache_proxy_balancer_response_kbytes_total{balancer="balancer://current",worker="http://app-03:8080"} 3105327
`
	if err := testutil.CollectAndCompare(e, strings.NewReader(want),
		"apache_proxy_balancer_request_kbytes_total",
		"apache_proxy_balancer_response_kbytes_total",
	); err != nil {
		t.Fatal(err)
	}
}
