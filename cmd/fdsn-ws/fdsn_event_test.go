package main

import (
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

	if s != " publicid = $1 AND latitude >= $2 AND latitude <= $3 AND longitude >= $4 AND longitude <= $5 AND depth > $6 AND depth < $7 AND magnitude > $8 AND magnitude < $9 AND origintime >= $10 AND origintime <= $11" {
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

	for _, s := range []string{"", "time", "time-asc", "magnitude", "magnitude-asc"} {
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

	// Test event in the DB.
	//publicid   |   latitude   |  longitude  |  depth   | magnitude |          origintime
	//-------------+--------------+-------------+----------+-----------+-------------------------------
	//2015p768477 | -40.57806609 | 176.3257242 | 23.28125 |       2.3 | 2015-10-12 08:05:01.717692+00

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
