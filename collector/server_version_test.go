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

// TestServerVersionServerTokens checks the ServerVersion values that Apache
// reports with the different ServerTokens settings.
func TestServerVersionServerTokens(t *testing.T) {
	for _, tc := range []struct {
		name       string
		version    string
		expVersion string
	}{
		{
			name:    "Prod",
			version: "Apache",
		},
		{
			name:    "Major",
			version: "Apache/2",
		},
		{
			name:    "Minor",
			version: "Apache/2.4",
		},
		{
			name:    "Minimal",
			version: "Apache/2.4.62",
			expVersion: `
# HELP apache_version Apache server version
# TYPE apache_version gauge
apache_version 2.04062
`,
		},
		{
			name:    "OS",
			version: "Apache/2.4.62 (Debian)",
			expVersion: `
# HELP apache_version Apache server version
# TYPE apache_version gauge
apache_version 2.04062
`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("ServerVersion: " + tc.version + "\nServerMPM: event\nTotal Accesses: 7\n"))
			}))
			defer server.Close()

			e := NewExporter(promslog.NewNopLogger(), &Config{ScrapeURI: server.URL})

			expected := tc.expVersion + `
# HELP apache_accesses_total Current total apache accesses (*)
# TYPE apache_accesses_total counter
apache_accesses_total 7
# HELP apache_info Apache version information
# TYPE apache_info gauge
apache_info{mpm="event",version="` + tc.version + `"} 1
`
			if err := testutil.CollectAndCompare(e, strings.NewReader(expected), "apache_version", "apache_accesses_total", "apache_info"); err != nil {
				t.Fatal(err)
			}
		})
	}
}
