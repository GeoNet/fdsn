package main

import (
	"fmt"
	"math"
	"net/url"
	"reflect"
	"testing"
	"time"
)

func TestEventV1Query(t *testing.T) {
	var v url.Values = make(map[string][]string)

	v.Set("eventid", "2015p768477")
	v.Set("minlatitude", "-41.0")
	v.Set("maxlatitude", "-40.0")
	v.Set("minlongitude", "177.0")
	v.Set("maxlongitude", "178.0")
	v.Set("mindepth", "12.0")
	v.Set("maxdepth", "32.0")
	v.Set("minmagnitude", "2.4")
	v.Set("maxmagnitude", "6.4")
	v.Set("starttime", "2015-01-12T12:12:12")
	v.Set("endtime", "2015-02-12T12:12:12")
	v.Set("orderby", "time")
	v.Set("latitude", fmt.Sprintf("%f", math.MaxFloat64))
	v.Set("longitude", fmt.Sprintf("%f", math.MaxFloat64))
	v.Set("maxradius", "180")
	v.Set("minradius", "0")

	e, err := parseEventV1(v)
	if err != nil {
		t.Error(err)
	}

	ex := fdsnEventV1{
		PublicID:     "2015p768477",
		MinLatitude:  -41.0,
		MaxLatitude:  -40.0,
		MinLongitude: 177.0,
		MaxLongitude: 178.0,
		MinDepth:     12.0,
		MaxDepth:     32.0,
		MinMagnitude: 2.4,
		MaxMagnitude: 6.4,
		OrderBy:      "time",
		Latitude:     math.MaxFloat64,
		Longitude:    math.MaxFloat64,
		MaxRadius:    180.0,
		MinRadius:    0.0,
		NoData:       204,
		Format:       "xml",
		EventType:    "*", // default value
	}

	ex.StartTime.Time, err = time.Parse(time.RFC3339Nano, "2015-01-12T12:12:12.000000000Z")
	if err != nil {
		t.Error(err)
	}

	ex.EndTime.Time, err = time.Parse(time.RFC3339Nano, "2015-02-12T12:12:12.000000000Z")
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(e, ex) {
		t.Error("e not equal to ex")
	}

	s, a := e.filter()

	if s != " publicid = $1 AND latitude >= $2 AND latitude <= $3 AND ST_X(ST_ShiftLongitude(ST_MakePoint(longitude,0.0))) >= ST_X(ST_ShiftLongitude(ST_MakePoint($4,0.0))) AND ST_X(ST_ShiftLongitude(ST_MakePoint(longitude,0.0))) <= ST_X(ST_ShiftLongitude(ST_MakePoint($5,0.0))) AND depth > $6 AND depth < $7 AND magnitude > $8 AND magnitude < $9 AND origintime >= $10 AND origintime <= $11" {
		t.Errorf("query string not correct got %s", s)
	}

	if len(a) != 11 {
		t.Errorf("expected 11 args got %d", len(a))
	}

	v.Set("extraParam", "is not allowed")
	_, err = parseEventV1(v)
	if err == nil {
		t.Error("should error for query with extra parameters")
	}
}

func TestEventV1OrderBy(t *testing.T) {
	var v url.Values = make(map[string][]string)

	for _, s := range []string{"time", "time-asc", "magnitude", "magnitude-asc"} {
		v.Set("orderby", s)
		_, err := parseEventV1(v)
		if err != nil {
			t.Error(err)
		}
	}

	v.Set("orderby", "unknown")
	_, err := parseEventV1(v)
	if err == nil {
		t.Error("should error for unknown orderby option.")
	}
}

func TestTimeParse(t *testing.T) {
	var tm Time

	if err := tm.UnmarshalText([]byte("2015-01-12T12:12:12.999999")); err != nil {
		t.Error(err)
	}

	if err := tm.UnmarshalText([]byte("2015-01-12T12:12:12")); err != nil {
		t.Error(err)
	}

	if err := tm.UnmarshalText([]byte("2015-01-12")); err != nil {
		t.Error(err)
	}

	if err := tm.UnmarshalText([]byte("2015-01-12T12:12:12-09:00")); err == nil {
		t.Error("expected an error for invalid time string.")
	}
}

func TestEventQuery(t *testing.T) {
	setup(t)
	defer teardown()

	vals := []struct {
		k string
		v string
	}{
		{"eventid", "2015p768477"},
		{"minlatitude", "-42.0"},
		{"maxlatitude", "-39.0"},
		{"minlongitude", "175.0"},
		{"maxlongitude", "177.0"},
		{"mindepth", "22.0"},
		{"maxdepth", "24.0"},
		{"minmagnitude", "2.0"},
		{"maxmagnitude", "6.0"},
		{"starttime", "2015-10-12T08:04:01"},
		{"endtime", "2015-10-12T08:06:01"},
		{"orderby", "time"},
		{"orderby", "time-asc"},
		{"orderby", "magnitude"},
		{"orderby", "magnitude-asc"},
		{"updatedafter", "2015-10-12T08:04:01"},
	}

	//var quakeml string
	var v url.Values

	// remake v in the loop to test each entry in vals independently.
	for _, q := range vals {
		v = make(map[string][]string)

		v.Set(q.k, q.v)
		e, err := parseEventV1(v)
		if err != nil {
			t.Error(err)
		}

		c, err := e.count()
		if err != nil {
			t.Error(err)
		}

		if c == 0 {
			q, a := e.filter()
			t.Errorf("should find at least 1 row for %+v %+v", q, a)
		}
	}

	// make v before the loop to test each entry in vals by adding it to a
	// growing filter chain.
	v = make(map[string][]string)

	for _, q := range vals {
		v.Set(q.k, q.v)
		e, err := parseEventV1(v)
		if err != nil {
			t.Error(err)
		}

		c, err := e.count()
		if err != nil {
			t.Error(err)
		}

		if c == 0 {
			q, a := e.filter()
			t.Errorf("should find at least 1 row for %+v %+v", q, a)
		}
	}
}

func TestEventGeomBounds(t *testing.T) {
	vals := []struct {
		k string
		v string
	}{
		{"minlatitude", "-91.0"},
		{"maxlatitude", "91.0"},
		{"minlongitude", "-181.0"},
		{"maxlongitude", "181.0"},
	}

	// remake v in the loop to test each entry in vals independently.
	var v url.Values

	for _, q := range vals {
		v = make(map[string][]string)

		v.Set(q.k, q.v)
		_, err := parseEventV1(v)
		if err == nil {
			t.Errorf("expected geom bounds error for %s %s", q.k, q.v)
		}
	}
}

func TestEventBoundingRadius(t *testing.T) {
	setup(t)
	defer teardown()

	v := url.Values{}

	// test against record: (-40.57806609, 176.3257242)
	v.Set("latitude", "-41.57806609") // 1 degree diff in lat
	v.Set("longitude", "176.3257242")

	// test range : < 2.0
	v.Set("maxradius", "2.0")
	e, err := parseEventV1(v)
	if err != nil {
		t.Error(err)
	}

	c, err := e.count()
	if err != nil {
		t.Error(err)
	}

	if c != 1 {
		t.Errorf("expected 1 record got %d.", c)
	}

	// test range 1.1 ~ 2.0
	v.Set("minradius", "1.1")
	e, err = parseEventV1(v)
	if err != nil {
		t.Error(err)
	}

	c, err = e.count()
	if err != nil {
		t.Error(err)
	}

	if c != 0 {
		t.Errorf("expected 0 record got %d.", c)
	}

	// test range 0.9 ~ 0.99
	v.Set("maxradius", "0.99")
	v.Set("minradius", "0.9")
	e, err = parseEventV1(v)
	if err != nil {
		t.Error(err)
	}

	c, err = e.count()
	if err != nil {
		t.Error(err)
	}

	if c != 0 {
		t.Errorf("expected 0 record got %d.", c)
	}
}

func TestEventAbbreviations(t *testing.T) {
	vals := []struct {
		k string
		v string
	}{
		{"minlat", "-38.0"},
		{"maxlat", "-46.0"},
		{"minlon", "-176.0"},
		{"maxlon", "-178.0"},
		{"lat", "-37.5"},
		{"lon", "-176.5"},
		{"minmag", "1.1"},
		{"maxmag", "2.2"},
		{"start", "2016-09-04T00:00:00"},
		{"end", "2016-09-05T00:00:00"},
	}

	// remake v in the loop to test each entry in vals independently.
	v := url.Values{}
	for _, q := range vals {
		v.Set(q.k, q.v)
	}

	e, err := parseEventV1(v)
	if err != nil {
		t.Error(err)
	}

	if e.MinLatitude != -38.0 {
		t.Errorf("expected -38.0 for minlat got %f.\n", e.MinLatitude)
	}
	if e.MaxLatitude != -46.0 {
		t.Errorf("expected -46.0 for maxlat got %f.\n", e.MaxLatitude)
	}
	if e.MinLongitude != -176.0 {
		t.Errorf("expected -176.0 for minlon got %f.\n", e.MinLongitude)
	}
	if e.MaxLongitude != -178.0 {
		t.Errorf("expected -178.0 for maxlon got %f.\n", e.MaxLongitude)
	}
	if e.Latitude != -37.5 {
		t.Errorf("expected -37.5 for lat got %f.\n", e.Latitude)
	}
	if e.Longitude != -176.5 {
		t.Errorf("expected -176.5 for lon got %f.\n", e.Longitude)
	}
	if e.MinMagnitude != 1.1 {
		t.Errorf("expected 1.1 for minmag %f.\n", e.MinMagnitude)
	}
	if e.MaxMagnitude != 2.2 {
		t.Errorf("expected 2.2 for maxmag %f.\n", e.MaxMagnitude)
	}

	tm, _ := time.Parse(time.RFC3339Nano, "2016-09-04T00:00:00.000000000Z")
	if !e.StartTime.Equal(tm) {
		t.Errorf("start parameter error: %s", e.StartTime.Format(time.RFC3339Nano))
	}

	tm, _ = time.Parse(time.RFC3339Nano, "2016-09-05T00:00:00.000000000Z")
	if !e.EndTime.Equal(tm) {
		t.Errorf("end parameter error: %s", e.EndTime.Format(time.RFC3339Nano))
	}
}

func TestLongitudeWrap180(t *testing.T) {
	setup(t)
	defer teardown()

	// test data: one at 176.3257242 and another at -176.3257242
	v := url.Values{}
	v.Set("minlon", "177.0")
	e, err := parseEventV1(v)
	if err != nil {
		t.Error(err)
	}

	c, err := e.count()
	if err != nil {
		t.Error(err)
	}

	if c != 1 {
		t.Errorf("expected 1 records got %d\n", c)
	}

	v = url.Values{}
	v.Set("minlon", "176")
	v.Set("maxlon", "-176")
	e, err = parseEventV1(v)
	if err != nil {
		t.Error(err)
	}

	c, err = e.count()
	if err != nil {
		t.Error(err)
	}

	if c != 2 {
		t.Errorf("expected 2 records got %d\n", c)
	}

	v = url.Values{}
	v.Set("maxlon", "-177.0")
	e, err = parseEventV1(v)
	if err != nil {
		t.Error(err)
	}

	c, err = e.count()
	if err != nil {
		t.Error(err)
	}

	if c != 1 {
		t.Errorf("expected 1 records got %d\n", c)
	}

	v = url.Values{}
	v.Set("lon", "179.4")
	v.Set("lat", "-40.57806609")
	v.Set("maxradius", "5")
	// 2 records, 176.3257242 and another at -176.3257242, distances are 4.27 and 3.07
	// if the query can cross 180 then we'll get 2 records
	e, err = parseEventV1(v)
	if err != nil {
		t.Error(err)
	}

	c, err = e.count()
	if err != nil {
		t.Error(err)
	}

	if c != 2 {
		t.Errorf("expected 2 records got %d\n", c)
	}
}

func TestEventTypes(t *testing.T) {
	queryCases := []struct {
		query     string
		shouldErr bool
		expected  []interface{}
	}{
		{"earthquake", false, []interface{}{"earthquake"}},
		{"e*", false, []interface{}{"earthquake", "explosion", "experimental explosion"}},
		{"z*", true, nil}, // no such match, expected value doesn't matter
		{"experimental explosion", false, []interface{}{"experimental explosion"}},
		{"e*,a*", false, []interface{}{"earthquake", "anthropogenic event", "explosion", "accidental explosion", "experimental explosion", "atmospheric event", "acoustic noise", "avalanche"}},
		// TODO: how do query for "unset eventtype"? The spec list all allowed values and empty is not in the list.
	}
	for _, c := range queryCases {
		v := url.Values{}
		v.Set("eventtype", c.query)
		e, err := parseEventV1(v)
		if !c.shouldErr && err != nil {
			t.Errorf("error %s: %v", c.query, err)
			continue
		}
		if c.shouldErr && err == nil {
			t.Errorf("expected to error but passed for %s", c.query)
			continue
		}
		if len(e.eventTypeSlice) != len(c.expected) {
			t.Errorf("expected %v got %v", c.expected, e.eventTypeSlice)
		}
		for i, v := range c.expected {
			if e.eventTypeSlice[i] != v {
				t.Errorf("expected %s got %s", v, e.eventTypeSlice[i])
			}
		}
	}

}
