// +build integration

package main

import (
	wt "github.com/GeoNet/kit/weft/wefttest"
	"testing"
)

var routesIntegration = wt.Requests{
	// fdsn-ws-dataselect
	{ID: wt.L(), URL: "/fdsnws/dataselect/1/query?starttime=2016-01-09T00:00:00&endtime=2016-01-09T23:00:00&network=NZ&station=CHST&location=01&channel=LOG", Content: "application/vnd.fdsn.mseed"},
	// abbreviated params
	{ID: wt.L(), URL: "/fdsnws/dataselect/1/query?start=2016-01-09T00:00:00&end=2016-01-09T23:00:00&net=NZ&sta=CHST&loc=01&cha=LOG", Content: "application/vnd.fdsn.mseed"},
}

// TestRoutesIntegration runs an integration test against S3
// Env var need to be set (see env.list).
// Run using:
// go test -tags integration -v -run TestRoutesIntegration
func TestRoutesIntegration(t *testing.T) {
	setup(t)
	setup(t)
	defer teardown()

	populateHoldings(t)

	for _, r := range routesIntegration {
		if b, err := r.Do(ts.URL); err != nil {
			t.Error(err)
			if len(b) > 0 {
				t.Error(string(b))
			}
		}
	}

	if err := routes.DoCheckQuery(ts.URL); err != nil {
		t.Error(err)
	}
}
