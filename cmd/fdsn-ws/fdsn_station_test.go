package main

import (
	"encoding/xml"
	"net/url"
	"strings"
	"testing"

	_ "github.com/GeoNet/fdsn/internal/fdsn"
	wt "github.com/GeoNet/kit/weft/wefttest"
)

// NOTE: To run the test, please export :
// STATION_XML_META_KEY=fdsn-station-test.xml

func TestStationFilter(t *testing.T) {
	var e fdsnStationV1Search
	var err error

	setup(t)
	defer teardown()
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

	// Test special case
	c = *fdsnStations.fdsn
	v = make(map[string][]string)
	v.Set("level", "network")
	v.Set("station", "ABCD")
	if e, err = parseStationV1(v); err != nil {
		t.Error(err)
	}
	c.doFilter([]fdsnStationV1Search{e})
	if len(c.Network) != 0 {
		t.Errorf("Incorrect filter result. Expected no record got %d", len(c.Network))
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

func TestFormatText(t *testing.T) {
	var e fdsnStationV1Search
	var err error
	var v url.Values = make(map[string][]string)

	c := *fdsnStations.fdsn

	v.Set("startTime", "2012-01-19T22:00:00")
	v.Set("station", "ARAZ")
	v.Set("channel", "EHZ")
	v.Set("level", "channel")
	if e, err = parseStationV1(v); err != nil {
		t.Error(err)
	}

	c.doFilter([]fdsnStationV1Search{e})
	b := c.marshalText(STATION_LEVEL_CHANNEL)
	exp := `#Network | Station | Location | Channel | Latitude | Longitude | Elevation | Depth | Azimuth | Dip | SensorDescription | Scale | ScaleFreq | ScaleUnits | SampleRate | StartTime | EndTime
NZ|ARAZ|10|EHZ|-38.627690|176.120060|420.000000|0.000000|0.000000|-90.000000|Short Period Seismometer|74574725.120000|15.000000|m/s|100.000000|2011-06-20T04:00:01|
`
	if b.String() != exp {
		t.Errorf("Incorrect text result.")
	}

	c = *fdsnStations.fdsn
	v.Set("level", "network")
	if e, err = parseStationV1(v); err != nil {
		t.Error(err)
	}

	c.doFilter([]fdsnStationV1Search{e})
	b = c.marshalText(STATION_LEVEL_NETWORK)
	exp = `#Network | Description | StartTime | EndTime | TotalStations
NZ|New Zealand National Seismograph Network|1884-02-01T00:00:00||2
`
	if b.String() != exp {
		t.Errorf("Incorrect text result.")
	}

	c = *fdsnStations.fdsn
	v.Set("level", "station")
	if e, err = parseStationV1(v); err != nil {
		t.Error(err)
	}

	c.doFilter([]fdsnStationV1Search{e})
	b = c.marshalText(STATION_LEVEL_STATION)
	exp = `#Network | Station | Latitude | Longitude | Elevation | SiteName | StartTime | EndTime
NZ|ARAZ|-38.627690|176.120060|420.000000|Aratiatia Landcorp Farm|2007-05-20T23:00:00|
`
	if b.String() != exp {
		t.Errorf("Incorrect text result.")
	}
}

// To profiling, you'll have to use full fdsn-station xml as data source:
//  1. Put full fdsn-station.xml in etc/.
//  2. export FDSN_STATION_XML_META_KEY=fdsn-station.xml
//  3. Run `go test -bench=StationQuery -benchmem -run=^$`.
//     Note: You must specify -run=^$ to skip test functions since you're not using test fdsn-station xml.
//
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

func TestPost(t *testing.T) {
	setup(t)
	defer teardown()

	// text format test
	body := `level=station
format=text
NZ ARA* * EHE*  2001-01-01T00:00:00 *
NZ ARH? * EHN*  2001-01-01T00:00:00 *`
	expected := strings.TrimSpace(`
#Network | Station | Latitude | Longitude | Elevation | SiteName | StartTime | EndTime
NZ|ARAZ|-38.627690|176.120060|420.000000|Aratiatia Landcorp Farm|2007-05-20T23:00:00|
NZ|ARHZ|-39.263100|176.995900|270.000000|Aropaoanui|2010-03-11T00:00:00|`)

	route := wt.Request{ID: wt.L(), URL: "/fdsnws/station/1/query", Method: "POST", PostBody: []byte(body), Content: "text/plain"}

	b, err := route.Do(ts.URL)
	if err != nil {
		t.Error(err)
	}

	if strings.TrimSpace(string(b)) != expected {
		t.Errorf("Unexpected query result.")
	}

	// xml format test
	body = `level=station
format=xml
NZ ARA* * EHE*  2001-01-01T00:00:00 *
NZ ARH? * EHN*  2001-01-01T00:00:00 *`
	testXml := `<?xml version="1.0" encoding="UTF-8"?>
	<FDSNStationXML schemaVersion="1" xmlns="http://www.fdsn.org/xml/station/1">
	<Source>GeoNet</Source>
	<Sender>WEL(GNS_Test)</Sender>
	<Module>Delta</Module>
	<Created>2017-09-26T02:37:17</Created>
	<Network code="NZ" startDate="1884-02-01T00:00:00" restrictedStatus="open">
		<Description>New Zealand National Seismograph Network</Description>
		<TotalNumberStations>2</TotalNumberStations>
		<SelectedNumberStations>2</SelectedNumberStations>
		<Station code="ARAZ" startDate="2007-05-20T23:00:00" restrictedStatus="closed">
			<Description>Private seismograph sites</Description>
			<Comment>
				<Value>Location is given in NZGD2000</Value>
			</Comment>
			<Latitude datum="NZGD2000">-38.62769</Latitude>
			<Longitude datum="NZGD2000">176.12006</Longitude>
			<Elevation>420</Elevation>
			<Site>
				<Name>Aratiatia Landcorp Farm</Name>
				<Description>9 km north of Taupo</Description>
			</Site>
			<CreationDate>2007-05-20T23:00:00</CreationDate>
			<TotalNumberChannels>9</TotalNumberChannels>
			<SelectedNumberChannels>3</SelectedNumberChannels>
		</Station>
		<Station code="ARHZ" startDate="2010-03-11T00:00:00" restrictedStatus="open">
			<Description>Hawke&#39;s Bay regional seismic network</Description>
			<Comment>
				<Value>Location is given in WGS84</Value>
			</Comment>
			<Latitude datum="WGS84">-39.2631</Latitude>
			<Longitude datum="WGS84">176.9959</Longitude>
			<Elevation>270</Elevation>
			<Site>
				<Name>Aropaoanui</Name>
				<Description>28 km north of Napier</Description>
			</Site>
			<CreationDate>2010-03-11T00:00:00</CreationDate>
			<TotalNumberChannels>6</TotalNumberChannels>
			<SelectedNumberChannels>2</SelectedNumberChannels>
		</Station>
	</Network>
	</FDSNStationXML>`
	var src FDSNStationXML

	err = xml.Unmarshal([]byte(testXml), &src)
	if err != nil {
		t.Fatal(err)
	}

	route = wt.Request{ID: wt.L(), URL: "/fdsnws/station/1/query", Method: "POST", PostBody: []byte(body), Content: "application/xml"}

	b, err = route.Do(ts.URL)
	if err != nil {
		t.Error(err)
	}

	var resp FDSNStationXML
	err = xml.Unmarshal(b, &resp)
	if err != nil {
		t.Error(err)
	}

	if len(src.Network) != len(resp.Network) {
		t.Errorf("expected Network length %d got %d", len(src.Network), len(resp.Network))
	}

	// Only check for some important values.
	for i := range src.Network {
		sn := src.Network[i]
		rn := resp.Network[i]

		if len(sn.Station) != len(rn.Station) {
			t.Errorf("expected Station length %d got %d", len(sn.Station), len(rn.Station))
		}

		for j := range sn.Station {
			ss := sn.Station[j]
			rs := rn.Station[j]

			if ss.Code != rs.Code {
				t.Errorf("station %d are not equal in Code", j)
			}
			if ss.StartDate != rs.StartDate {
				t.Errorf("station %d are not equal in StartData", j)
			}
			if ss.Longitude.Value != rs.Longitude.Value {
				t.Errorf("station %d are not equal in Longitude", j)
			}
			if ss.Latitude.Value != rs.Latitude.Value {
				t.Errorf("station %d are not equal in Latitude", j)
			}
			if ss.Elevation.Value != rs.Elevation.Value {
				t.Errorf("station %d are not equal in Elevation", j)
			}
			if ss.TotalNumberChannels != rs.TotalNumberChannels {
				t.Errorf("station %d are not equal in TotalNumberChannels", j)
			}
			if ss.SelectedNumberChannels != rs.SelectedNumberChannels {
				t.Errorf("station %d are not equal in SelectedNumberChannels", j)
			}
		}
	}
}

// test filter. Especially for nodes having no children.
func TestDoFilter(t *testing.T) {
	var err error
	var fdsn FDSNStationXML
	var query url.Values
	var hasValue bool

	setup(t)
	defer teardown()
	//
	// basic case
	//

	// Test network filter - match case
	query = make(map[string][]string)
	fdsn = makeTestFDSN("NZ", "STA1", "10", "CHA1")
	query.Set("network", "NZ")
	if hasValue, err = testCase(&fdsn, query); err != nil {
		t.Error(err)
	}
	if !hasValue {
		t.Errorf("expected to have NZ network got empty")
	}

	// Test network filter - unmatch case
	query = make(map[string][]string)
	fdsn = makeTestFDSN("NZ", "STA1", "10", "CHA1")
	query.Set("network", "MC")
	if hasValue, err = testCase(&fdsn, query); err != nil {
		t.Error(err)
	}
	if hasValue {
		t.Errorf("expected to be empty got %v", fdsn)
	}

	// Test station filter - match case
	query = make(map[string][]string)
	fdsn = makeTestFDSN("NZ", "STA1", "10", "CHA1")
	query.Set("network", "NZ")
	query.Set("station", "STA1")
	if hasValue, err = testCase(&fdsn, query); err != nil {
		t.Error(err)
	}
	if !hasValue {
		t.Errorf("expected to have STA1 station got empty")
	}

	// Test station filter - unmatch case
	query = make(map[string][]string)
	fdsn = makeTestFDSN("NZ", "STA1", "10", "CHA1")
	query.Set("network", "NZ")
	query.Set("station", "STA2")
	if hasValue, err = testCase(&fdsn, query); err != nil {
		t.Error(err)
	}
	if hasValue {
		t.Errorf("expected to be empty got %v", fdsn)
	}

	// Test channel filter - match case
	query = make(map[string][]string)
	fdsn = makeTestFDSN("NZ", "STA1", "10", "CHA1")
	query.Set("network", "NZ")
	query.Set("station", "STA1")
	query.Set("channel", "CHA1")
	if hasValue, err = testCase(&fdsn, query); err != nil {
		t.Error(err)
	}
	if !hasValue {
		t.Errorf("expected to have CHA1 channel got empty")
	}

	// Test channel filter - unmatch case
	query = make(map[string][]string)
	fdsn = makeTestFDSN("NZ", "STA1", "10", "CHA1")
	query.Set("network", "NZ")
	query.Set("station", "STA1")
	query.Set("channel", "CHA2")
	if hasValue, err = testCase(&fdsn, query); err != nil {
		t.Error(err)
	}
	if hasValue {
		t.Errorf("expected to be empty got %v", fdsn)
	}

	//
	// complicated cases
	//

	// network without stations, asking level station, returns till network level
	query = make(map[string][]string)
	fdsn = makeTestFDSN("NZ", "", "", "")
	query.Set("network", "NZ")
	query.Set("level", "station")
	if hasValue, err = testCase(&fdsn, query); err != nil {
		t.Error(err)
	}
	if !hasValue {
		t.Errorf("expected to have NZ network got empty")
	}
	if len(fdsn.Network) != 1 || len(fdsn.Network[0].Station) != 0 {
		t.Errorf("exepcted to have NZ network only, got %v", fdsn)
	}

	// network without stations, query contains station, should fail
	query = make(map[string][]string)
	fdsn = makeTestFDSN("NZ", "", "", "")
	query.Set("network", "NZ")
	query.Set("network", "STA1")
	if hasValue, err = testCase(&fdsn, query); err != nil {
		t.Error(err)
	}
	if hasValue {
		t.Errorf("expected to be emptyu got %v", fdsn)
	}

	// station without channels, asking level channel, returns till station level
	query = make(map[string][]string)
	fdsn = makeTestFDSN("NZ", "STA1", "", "")
	query.Set("network", "NZ")
	query.Set("station", "STA1")
	query.Set("level", "channel")
	if hasValue, err = testCase(&fdsn, query); err != nil {
		t.Error(err)
	}
	if !hasValue {
		t.Errorf("expected to have STA1 station got empty")
	}
	if len(fdsn.Network) != 1 || len(fdsn.Network[0].Station) != 1 || len(fdsn.Network[0].Station[0].Channel) != 0 {
		t.Errorf("exepcted to have NZ/STA1, got %v", fdsn)
	}

	// station without channels, query contains channel, should fail
	query = make(map[string][]string)
	fdsn = makeTestFDSN("NZ", "STA1", "", "")
	query.Set("network", "NZ")
	query.Set("station", "STA1")
	query.Set("channel", "CHA1")
	if hasValue, err = testCase(&fdsn, query); err != nil {
		t.Error(err)
	}
	if hasValue {
		t.Errorf("expected to be emptyu got %v", fdsn)
	}

	// two station without children, we should see both
	query = make(map[string][]string)
	fdsn = makeTestFDSN("NZ", "STA1", "", "")
	fdsn.Network[0].Station = append(fdsn.Network[0].Station, makeTestStation("STA2"))
	query.Set("network", "NZ")
	query.Set("station", "STA1,STA2")
	query.Set("channel", "*")
	if hasValue, err = testCase(&fdsn, query); err != nil {
		t.Error(err)
	}
	if !hasValue {
		t.Errorf("expected to be values got empty result")
	}
	// we should get 2 results
	if len(fdsn.Network[0].Station) != 2 {
		t.Errorf("expected to be 2 stations got %v", fdsn)
	}
}

// helper functions
func testCase(c *FDSNStationXML, query url.Values) (bool, error) {
	var e fdsnStationV1Search
	var err error
	if e, err = parseStationV1(query); err != nil {
		return false, err
	}
	return c.doFilter([]fdsnStationV1Search{e}), nil
}

var emptyXsdDatetime = xsdDateTime(emptyDateTime)

func makeTestFDSN(network, station, location, channel string) FDSNStationXML {
	var c = FDSNStationXML{
		Network: []NetworkType{
			{
				BaseNodeType: BaseNodeType{
					Code:      network,
					StartDate: emptyXsdDatetime,
					EndDate:   emptyXsdDatetime,
				},
				Station: []StationType{},
			},
		},
	}

	if station != "" {
		c.Network[0].Station = append(c.Network[0].Station, makeTestStation(station))

		if channel != "" {
			c.Network[0].Station[0].Channel = append(c.Network[0].Station[0].Channel, makeTestChannel(channel, location))
		}
	}

	return c
}

func makeTestStation(code string) StationType {
	return StationType{
		BaseNodeType: BaseNodeType{
			Code:      code,
			StartDate: emptyXsdDatetime,
			EndDate:   emptyXsdDatetime,
		},
		Channel: []ChannelType{},
	}
}

func makeTestChannel(code, locationCode string) ChannelType {
	return ChannelType{
		BaseNodeType: BaseNodeType{
			Code:      code,
			StartDate: emptyXsdDatetime,
			EndDate:   emptyXsdDatetime,
		},
		LocationCode: locationCode,
	}
}
