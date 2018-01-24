package main

import (
	"net/url"
	"testing"
)

// NOTE: To run the test, please export :
// STATION_XML_META_KEY=fdsn-station-test.xml

func init() {
}

func TestStationFilter(t *testing.T) {
	var e fdsnStationV1Search
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

	c := *fdsnStations.fdsn
	c.doFilter([]fdsnStationV1Search{e})

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
	c.doFilter([]fdsnStationV1Search{e})

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
	c.doFilter([]fdsnStationV1Search{e})

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
	c.doFilter([]fdsnStationV1Search{e})

	if len(c.Network) != 1 {
		t.Errorf("Incorrect filter result. No valid record.")
	}

	if len(c.Network[0].Station) != 1 {
		t.Errorf("Incorrect filter result. Expect 1 got %d", len(c.Network[0].Station))
	}

	// Channel code matching
	c = *fdsnStations.fdsn
	v.Set("station", "ARAZ")
	v.Set("channel", "EHE")
	v.Set("level", "response")

	if e, err = parseStationV1(v); err != nil {
		t.Error(err)
	}
	c.doFilter([]fdsnStationV1Search{e})

	if len(c.Network) != 1 {
		t.Errorf("Incorrect filter result. No valid record.")
	}

	if len(c.Network[0].Station) != 1 {
		t.Errorf("Incorrect filter result. Expect 1 got %d", len(c.Network[0].Station))
	}

	if len(c.Network[0].Station[0].Channel) != 3 {
		t.Errorf("Incorrect filter result. Expect 3 got %d", len(c.Network[0].Station[0].Channel))
	}

	if len(c.Network[0].Station[0].Channel[0].Response.Stage) != 3 {
		t.Errorf("Incorrect filter result. Expect 3 got %d", len(c.Network[0].Station[0].Channel[0].Response.Stage))
	}

	// check trimming
	v.Set("level", "channel")

	if e, err = parseStationV1(v); err != nil {
		t.Error(err)
	}
	c.doFilter([]fdsnStationV1Search{e})

	if c.Network[0].Station[0].Channel[0].Response.Stage != nil {
		t.Errorf("Incorrect filter result. Expect nil got %d", len(c.Network[0].Station[0].Channel[0].Response.Stage))
	}

	// latlng radius matching
	c = *fdsnStations.fdsn
	v = make(map[string][]string)
	v.Set("latitude", "-38.6")
	v.Set("longitude", "176.1")
	v.Set("maxradius", "0.5")
	v.Set("level", "channel")

	if e, err = parseStationV1(v); err != nil {
		t.Error(err)
	}
	c.doFilter([]fdsnStationV1Search{e})
	if len(c.Network) != 1 {
		t.Errorf("Incorrect filter result. No valid record.")
	}

	if len(c.Network[0].Station) != 1 {
		t.Errorf("Incorrect filter result. Expect 1 got %d", len(c.Network[0].Station))
	}

	if len(c.Network[0].Station[0].Channel) != 9 {
		t.Errorf("Incorrect filter result. Expect 6 got %d", len(c.Network[0].Station[0].Channel))
	}

	// test if minradius works
	v.Set("minradius", "0.1")
	if e, err = parseStationV1(v); err != nil {
		t.Error(err)
	}
	c.doFilter([]fdsnStationV1Search{e})
	if len(c.Network) != 0 {
		t.Errorf("Incorrect filter result. Expected no record but got %d.", len(c.Network))
	}

	// Simulates POST Test.
	// The spaces between fields should be ignored.
	postBody := `level=   channel
		NZ ARA* * EHE*      2001-01-01T00:00:00 *
		NZ ARH? * EHN*  2001-01-01T00:00:00 *`
	var vs []fdsnStationV1Search
	if vs, err = parseStationV1Post(postBody); err != nil {
		t.Error(err)
	}

	c = *fdsnStations.fdsn
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

/*
Testdata timeline

Station 1:
2007-05-20T23   2011-03-06T22   2011-06-20T04
     |------3 cha----|
                      |-----3 cha----|
                                      |-----3 cha----------->
Station 2:
                     2010-03-11T21   2012-01-19T22
                          |-----3 cha----|
                                          |-----3 cha------->
*/
func TestStartEnd(t *testing.T) {
	var e fdsnStationV1Search
	var err error
	var v url.Values = make(map[string][]string)

	c := *fdsnStations.fdsn

	// This should filter out 3 channels end at 2011-03-06T22:00:00
	v.Set("startTime", "2011-03-06T22:00:01")
	v.Set("level", "channel")

	if e, err = parseStationV1(v); err != nil {
		t.Error(err)
	}

	c.doFilter([]fdsnStationV1Search{e})

	if len(c.Network) != 1 {
		t.Errorf("Incorrect filter result. No valid record.")
	}

	if len(c.Network[0].Station) != 2 {
		t.Errorf("Incorrect filter result. Expect 2 got %d", len(c.Network[0].Station))
	}

	// 1st channel, 6
	if len(c.Network[0].Station[0].Channel) != 6 {
		t.Errorf("Incorrect filter result. Expect 6 got %d", len(c.Network[0].Station[0].Channel))
	}

	// 2nd channel, 6
	if len(c.Network[0].Station[1].Channel) != 6 {
		t.Errorf("Incorrect filter result. Expect 6 got %d", len(c.Network[0].Station[1].Channel))
	}

	c = *fdsnStations.fdsn // reset data
	v = make(map[string][]string)
	v.Set("startAfter", "2007-05-20T23:00:00") // station1's latter 6 channels (skip check station2)
	v.Set("level", "channel")

	if e, err = parseStationV1(v); err != nil {
		t.Error(err)
	}

	c.doFilter([]fdsnStationV1Search{e})

	if len(c.Network[0].Station[0].Channel) != 6 {
		t.Errorf("Incorrect filter result. Expect 6 got %d", len(c.Network[0].Station[0].Channel))
	}

	c = *fdsnStations.fdsn // reset data
	v = make(map[string][]string)
	v.Set("startBefore", "2007-05-20T23:00:01") // only station1's first 3 channels
	v.Set("level", "channel")
	if e, err = parseStationV1(v); err != nil {
		t.Error(err)
	}

	c.doFilter([]fdsnStationV1Search{e})

	if len(c.Network[0].Station[0].Channel) != 3 {
		t.Errorf("Incorrect filter result. Expect 3 got %d", len(c.Network[0].Station[0].Channel))
	}

	c = *fdsnStations.fdsn // reset data
	v = make(map[string][]string)
	// This sould only include left 6 channels of station1.
	v.Set("startBefore", "2011-06-20T04:00:00") // station1=6, station2=3
	v.Set("endTime", "2010-03-11T00:00:00")     // station1=3
	v.Set("level", "channel")

	if e, err = parseStationV1(v); err != nil {
		t.Error(err)
	}

	c.doFilter([]fdsnStationV1Search{e})

	if len(c.Network) != 1 {
		t.Errorf("Incorrect filter result. No valid record.")
	}

	if len(c.Network[0].Station) != 1 {
		t.Errorf("Incorrect filter result. Expect 2 got %d", len(c.Network[0].Station))
	}

	if len(c.Network[0].Station[0].Channel) != 3 {
		t.Errorf("Incorrect filter result. Expect 6 got %d", len(c.Network[0].Station[0].Channel))
	}

	c = *fdsnStations.fdsn // reset data
	v = make(map[string][]string)
	v.Set("startTime", "2011-06-20T03:00:00") // station1= latter 6, station2=6
	v.Set("endAfter", "2011-06-20T05:00:00")  // station1=3, station2=6
	v.Set("level", "channel")

	if e, err = parseStationV1(v); err != nil {
		t.Error(err)
	}

	c.doFilter([]fdsnStationV1Search{e})

	if len(c.Network) != 1 {
		t.Errorf("Incorrect filter result. No valid record.")
	}

	if len(c.Network[0].Station) != 2 {
		t.Errorf("Incorrect filter result. Expect 2 got %d", len(c.Network[0].Station))
	}

	if len(c.Network[0].Station[0].Channel) != 3 {
		t.Errorf("Incorrect filter result. Expect 3 got %d", len(c.Network[0].Station[0].Channel))
	}

	if len(c.Network[0].Station[1].Channel) != 6 {
		t.Errorf("Incorrect filter result. Expect 6 got %d", len(c.Network[0].Station[1].Channel))
	}

	c = *fdsnStations.fdsn // reset data
	v = make(map[string][]string)
	v.Set("startTime", "2011-06-20T03:00:00") // station1= latter 6, station2=6
	v.Set("endBefore", "2011-06-20T04:00:01") // station1=6, station2=0
	v.Set("level", "channel")
	if e, err = parseStationV1(v); err != nil {
		t.Error(err)
	}

	c.doFilter([]fdsnStationV1Search{e})

	if len(c.Network[0].Station) != 1 {
		t.Errorf("Incorrect filter result. Expect 1 got %d", len(c.Network[0].Station))
	}

	if len(c.Network[0].Station[0].Channel) != 3 {
		t.Errorf("Incorrect filter result. Expect 3 got %d", len(c.Network[0].Station[0].Channel))
	}
}

// To profiling, you'll have to use full fdsn-station xml as data source:
// 1. Put full fdsn-station.xml in etc/.
// 2. export FDSN_STATION_XML_META_KEY=fdsn-station.xml
// 3. Run `go test -bench=StationQuery -benchmem -run=^$`.
//    Note: You must specify -run=^$ to skip test functions since you're not using test fdsn-station xml.
// Currently the benchmark result for my MacBookPro 2017 is:
// BenchmarkStationQuery/post-4               20000             74472 ns/op           78376 B/op        706 allocs/op
func BenchmarkStationQuery(b *testing.B) {
	postBody := `level=   channel
		NZ ARA* * EHE*      2001-01-01T00:00:00 *
		`

	params, err := parseStationV1Post(postBody)

	if err != nil {
		b.Error(err)
	}

	benchmarks := []struct {
		name   string
		params []fdsnStationV1Search
	}{
		{"post", params},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				c := *fdsnStations.fdsn
				c.doFilter(bm.params)
			}
		})
	}
}
