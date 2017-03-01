package main

import (
	wt "github.com/GeoNet/weft/wefttest"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

// NOTE: To run the test, please export :
// FDSN_STATION_XML_META_KEY=fdsn-station-test.xml

func TestStationV1Query(t *testing.T) {
	ts = httptest.NewServer(mux)
	defer ts.Close()

	wt.Request{Accept: "application/xml", URL: "/fdsnws/station/1/version"}.Do(ts.URL)
	wt.Request{Accept: "application/xml", URL: "/fdsnws/station/1/application.wadl"}.Do(ts.URL)

	for !stationsLoaded {
		time.Sleep(100)
	}

	wt.Request{Accept: "application/xml", URL: "/fdsnws/station/1/query"}.Do(ts.URL)
	wt.Request{Accept: "application/xml", URL: "/fdsnws/station/1/query?level=channel&starttime=1900-01-01T00:00:00"}.Do(ts.URL)
	wt.Request{Accept: "application/xml", URL: "/fdsnws/station/1/query?minlat=-41&maxlon=177"}.Do(ts.URL)

}

func TestStationFilter(t *testing.T) {
	for !stationsLoaded {
		time.Sleep(100)
	}

	c := fdsnStations
	if len(c.Network) != 1 {
		t.Errorf("Incorrect records for test data. No valid record.")
	}

	if len(c.Network[0].Station) != 2 {
		t.Errorf("Incorrect records for test data. Expect 2 got %d", len(c.Network[0].Station))
	}

	var e fdsnStationV1Parm
	var err error

	// Filter test
	var v url.Values = make(map[string][]string)

	// Lat/lng range
	v.Set("minlatitude", "-45.0")
	v.Set("maxlatitude", "-35.0")
	v.Set("minlongitude", "173.0")
	v.Set("maxlongitude", "177.0")

	if e, err = parseStationV1(v); err != nil {
		t.Error(err)
	}

	c.doFilter([]fdsnStationV1Parm{e})

	if len(c.Network) != 1 {
		t.Errorf("Incorrect filter result. No valid record.")
	}

	if len(c.Network[0].Station) != 2 {
		t.Errorf("Incorrect filter result. Expect 2 got %d", len(c.Network[0].Station))
	}

	// Time range
	v.Set("starttime", "2003-01-01T00:00:00")
	v.Set("end", "2011-12-31T00:00:00")
	if e, err = parseStationV1(v); err != nil {
		t.Error(err)
	}
	c.doFilter([]fdsnStationV1Parm{e})

	if len(c.Network) != 1 {
		t.Errorf("Incorrect filter result. No valid record.")
	}

	if len(c.Network[0].Station) != 2 {
		t.Errorf("Incorrect filter result. Expect 2 got %d", len(c.Network[0].Station))
	}

	// Station code matching
	v.Set("station", "AR*") // result ARAZ, ARHZ

	if e, err = parseStationV1(v); err != nil {
		t.Error(err)
	}
	c.doFilter([]fdsnStationV1Parm{e})

	if len(c.Network) != 1 {
		t.Errorf("Incorrect filter result. No valid record.")
	}

	if len(c.Network[0].Station) != 2 {
		t.Errorf("Incorrect filter result. Expect 2 got %d", len(c.Network[0].Station))
	}

	v.Set("station", "A?HZ") // result ARHZ
	if e, err = parseStationV1(v); err != nil {
		t.Error(err)
	}
	c.doFilter([]fdsnStationV1Parm{e})

	if len(c.Network) != 1 {
		t.Errorf("Incorrect filter result. No valid record.")
	}

	if len(c.Network[0].Station) != 1 {
		t.Errorf("Incorrect filter result. Expect 1 got %d", len(c.Network[0].Station))
	}

	// Channel code matching
	c = fdsnStations
	v.Set("station", "ARAZ")
	v.Set("channel", "EHE")
	v.Set("level", "channel")

	if e, err = parseStationV1(v); err != nil {
		t.Error(err)
	}
	c.doFilter([]fdsnStationV1Parm{e})

	if len(c.Network) != 1 {
		t.Errorf("Incorrect filter result. No valid record.")
	}

	if len(c.Network[0].Station) != 1 {
		t.Errorf("Incorrect filter result. Expect 1 got %d", len(c.Network[0].Station))
	}

	if len(c.Network[0].Station[0].Channel) != 3 {
		t.Errorf("Incorrect filter result. Expect 3 got %d", len(c.Network[0].Station[0].Channel))
	}

	// Simulates POST Test.
	// The spaces between fields should be ignored.
	postBody := `level=   channel
		NZ ARA* * EHE*      2001-01-01T00:00:00 *
		NZ ARH? * EHN*  2001-01-01T00:00:00 *`
	var vs []fdsnStationV1Parm
	if vs, err = parseStationV1Post(postBody); err != nil {
		t.Error(err)
	}

	c = fdsnStations
	c.doFilter(vs)

	if len(c.Network) != 1 {
		t.Errorf("Incorrect filter result. No valid record.")
	}

	if len(c.Network[0].Station) != 2 {
		t.Errorf("Incorrect filter result. Expect 2 got %d", len(c.Network[0].Station))
	}

	if len(c.Network[0].Station[0].Channel) != 3 {
		t.Errorf("Incorrect filter result. Expect 3 got %d", len(c.Network[0].Station[0].Channel))
	}

}
