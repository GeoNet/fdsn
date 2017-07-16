package main

import (
	"fmt"
	"github.com/GeoNet/fdsn/internal/holdings"
	wt "github.com/GeoNet/fdsn/internal/weft/wefttest"
	"net/http"
	"testing"
	"time"
)

// setup() adds event 2015p768477 to the DB.
var routes = wt.Requests{
	{ID: wt.L(), URL: "/sc3ml?eventid=2015p768477", Content: "application/xml"},

	// fdsn-ws-event
	{ID: wt.L(), URL: "/fdsnws/event/1", Content: "text/html"},
	{ID: wt.L(), URL: "/fdsnws/event/1/query?eventid=2015p768477", Content: "application/xml"},
	{ID: wt.L(), URL: "/fdsnws/event/1/version", Content: "text/plain"},
	{ID: wt.L(), URL: "/fdsnws/event/1/catalogs", Content: "application/xml"},
	{ID: wt.L(), URL: "/fdsnws/event/1/contributors", Content: "application/xml"},
	{ID: wt.L(), URL: "/fdsnws/event/1/application.wadl", Content: "application/xml"},

	// fdsn-ws-dataselect
	{ID: wt.L(), URL: "/fdsnws/dataselect/1", Content: "text/html"},
	{ID: wt.L(), URL: "/fdsnws/dataselect/1/version", Content: "text/plain"},
	// an invalid network or no files matching query should give 404 (could also give 204 as per spec)
	{ID: wt.L(), URL: "/fdsnws/dataselect/1/query?starttime=2016-01-09T00:00:00&endtime=2016-01-09T23:00:00&network=INVALID_NETWORK&station=CHST&location=01&channel=LOG",
		Content: "text/plain; charset=utf-8",
		Status:  http.StatusNoContent},
	// very old time range, no files:
	{ID: wt.L(), URL: "/fdsnws/dataselect/1/query?starttime=1900-01-09T00:00:00&endtime=1900-01-09T01:00:00&network=NZ&station=CHST&location=01&channel=LOG",
		Content: "text/plain; charset=utf-8",
		Status:  http.StatusNoContent},
	//{ID: wt.L(), URL: "/fdsnws/dataselect/1/query", Content: "text/plain", Status: http.StatusRequestEntityTooLarge},
	{ID: wt.L(), URL: "/fdsnws/dataselect/1/application.wadl", Content: "application/xml"},

	// fdsn-ws-station
	{ID: wt.L(), URL: "/fdsnws/station/1/version", Content: "text/plain"},
	{ID: wt.L(), URL: "/fdsnws/station/1/application.wadl", Content: "application/xml"},
	{ID: wt.L(), URL: "/fdsnws/station/1/query", Content: "application/xml"},
	{ID: wt.L(), URL: "/fdsnws/station/1/query?level=channel&starttime=1900-01-01T00:00:00", Content: "application/xml"},
	{ID: wt.L(), URL: "/fdsnws/station/1/query?level=channel&starttime=1900-01-01", Content: "application/xml"},
	{ID: wt.L(), URL: "/fdsnws/station/1/query?level=channel&starttime=1900-01-01T00:00:00.123", Content: "application/xml"},
	{ID: wt.L(), URL: "/fdsnws/station/1/query?level=channel&starttime=1900-01-01T00:00:00.123456", Content: "application/xml"},
	{ID: wt.L(), URL: "/fdsnws/station/1/query?level=channel&starttime=1900-01-01T00:00:00.1234567", Content: "text/plain", Status: http.StatusBadRequest},
	{ID: wt.L(), URL: "/fdsnws/station/1/query?level=channel&starttime=1900-01-01T00:00:0", Content: "text/plain", Status: http.StatusBadRequest},
	{ID: wt.L(), URL: "/fdsnws/station/1/query?level=channel&starttime=1900-01-0", Content: "text/plain", Status: http.StatusBadRequest},
	{ID: wt.L(), URL: "/fdsnws/station/1/query?minlat=-41&maxlon=177", Content: "application/xml"},
	{ID: wt.L(), URL: "/fdsnws/station/1/query?level=channel&starttime=1900-01-01T00:00:00&format=text", Content: "text/plain"},
	{ID: wt.L(), URL: "/fdsnws/station/1/query?format=y", Content: "text/plain", Status: http.StatusBadRequest},
	{ID: wt.L(), URL: "/fdsnws/station/1/query?net=*&level=network&format=xml", Content: "application/xml"},
	{ID: wt.L(), URL: "/fdsnws/station/1/query?lat=-38.6&lon=176.1", Content: "application/xml"},
	{ID: wt.L(), URL: "/fdsnws/station/1/query?lat=-38.6&lon=176.1&maxradius=1.0", Content: "application/xml"},
	{ID: wt.L(), URL: "/fdsnws/station/1/query?lat=-38.6&lon=176.1&maxradius=1.0&minradius=0.1", Content: "application/xml"},
	// supporting the includeavailability parameter is optional.  Some clients send the value `false` which is the default.
	// allow for this by ignoring includeavailability=false
	{ID: wt.L(), URL: "/fdsnws/station/1/query?net=*&level=network&format=xml&includeavailability=false", Content: "application/xml"},
	{ID: wt.L(), URL: "/fdsnws/station/1/query?net=*&level=network&format=xml&includeavailability=true", Content: "text/plain", Status: http.StatusBadRequest},
}

// Test all routes give the expected response.  Also check with
// cache busters and extra query paramters.
func TestRoutes(t *testing.T) {
	setup(t)
	setup(t)
	defer teardown()

	populateHoldings(t)

	for _, r := range routes {
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

// populateDb inserts a thwack of holdings that exist in the bucket so we can run our integration tests against them
func populateHoldings(t *testing.T) {
	stations := []string{"ALRD", "ALRZ", "CHST"}
	locations := []string{"01", "10"}
	channels := []string{"EHN", "VEP", "VKI", "LOG"}

	t1Str := "2016-01-01T00:00:00Z"
	t1, err := time.Parse(time.RFC3339, t1Str)
	if err != nil {
		t.Fatal(err)
	}

	t2Str := "2016-01-10T00:00:00Z"
	t2, err := time.Parse(time.RFC3339, t2Str)
	if err != nil {
		t.Fatal(err)
	}

	for _, sta := range stations {
		for _, cha := range channels {
			for _, loc := range locations {
				for step := t1; step.Before(t2); step = step.Add(time.Hour * 24) {
					h := holding{
						key: fmt.Sprintf("%d/NZ/%s/%s.D/NZ.%s.%s.%s.D.%d.%03d", step.Year(), sta, cha, sta, loc, cha, step.Year(), step.YearDay()),
						Holding: holdings.Holding{
							Network:    "NZ",
							Station:    sta,
							Channel:    cha,
							Location:   loc,
							Start:      step,
							NumSamples: 500000, // incorrect but we're just faking it
						},
					}

					err = h.save()
					if err != nil {
						t.Fatal(err)
					}
				}
			}
		}
	}
}
