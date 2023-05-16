package main

import (
	"bytes"
	"database/sql"
	"io/ioutil"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

var versions = []string{"2015p768477_0.7.xml", "2015p768477_0.8.xml", "2015p768477_0.9.xml", "2015p768477_0.10.xml", "2015p768477_0.11.xml"}

func TestEventUnmarshal(t *testing.T) {
	for _, input := range versions {
		b, err := ioutil.ReadFile("etc/" + input)
		if err != nil {
			t.Fatal(err)
		}

		var e event

		if err = unmarshal(b, &e); err != nil {
			t.Error(err)
		}

		if !strings.HasPrefix(e.Quakeml12Event, `<event publicID="smi:nz.org.geonet/2015p768477">`) {
			t.Errorf("%s: quakeml fragment should start with <event...", input)
		}

		if !strings.HasSuffix(e.Quakeml12Event, `</event>`) {
			t.Errorf("%s: quakeml fragment should end with </event>", input)
		}

		c := event{
			PublicID:              "2015p768477",
			EventType:             "earthquake",
			Longitude:             176.3257242,
			Latitude:              -40.57806609,
			Depth:                 23.28125,
			EvaluationMethod:      "NonLinLoc",
			EarthModel:            "nz3drx",
			EvaluationMode:        "manual",
			EvaluationStatus:      "confirmed",
			UsedPhaseCount:        44,
			UsedStationCount:      32,
			OriginError:           0.5592857863,
			AzimuthalGap:          166.4674465,
			MinimumDistance:       0.1217162272,
			Magnitude:             5.691131913,
			MagnitudeUncertainty:  0,
			MagnitudeType:         "M",
			MagnitudeStationCount: 171,
			Deleted:               false,
			Sc3ml:                 string(b),
		}

		c.ModificationTime, _ = time.Parse(time.RFC3339Nano, "2015-10-12T22:46:41.228824Z")
		c.OriginTime, _ = time.Parse(time.RFC3339Nano, "2015-10-12T08:05:01.717692Z")

		if c.Quakeml12Event, err = toQuakeMLEvent(b); err != nil {
			t.Error(err)
		}

		if !reflect.DeepEqual(e, c) {
			t.Errorf("c not equal to e, expected: %+v", e)
		}
	}
}

// TestEventType tests that the remapping of SC3ML event type to QuakeML is correct.
// The bug in the sc3ml_*_quakeml_1.2.xsl conversion (inserting "other" instead of "other event"
// has been fixed locally and reported upstream.  GMC 12 Sept 2017
func TestEventType(t *testing.T) {
	testCases := []struct {
		id        string
		version   []byte
		eventType []byte
	}{
		{id: loc(), version: sc3ml07, eventType: []byte("<type>not locatable</type>")},
		{id: loc(), version: sc3ml07, eventType: []byte("<type>outside of network interest</type>")},
		{id: loc(), version: sc3ml07, eventType: []byte("<type>duplicate</type>")},
		{id: loc(), version: sc3ml08, eventType: []byte("<type>not locatable</type>")},
		{id: loc(), version: sc3ml08, eventType: []byte("<type>outside of network interest</type>")},
		{id: loc(), version: sc3ml08, eventType: []byte("<type>duplicate</type>")},
		{id: loc(), version: sc3ml09, eventType: []byte("<type>not locatable</type>")},
		{id: loc(), version: sc3ml09, eventType: []byte("<type>outside of network interest</type>")},
		{id: loc(), version: sc3ml09, eventType: []byte("<type>duplicate</type>")},
	}

	for _, v := range testCases {

		// input test file is sc3ml 0.7 change the version string below to test each
		// sc3ml version that is supported.
		for _, input := range versions {
			b, err := ioutil.ReadFile("etc/" + input)
			if err != nil {
				t.Fatal(err)
			}

			b = bytes.Replace(b, sc3ml07, v.version, -1)
			b = bytes.Replace(b, []byte("<type>earthquake</type>"), v.eventType, -1)

			var e string
			if e, err = toQuakeMLEvent(b); err != nil {
				t.Errorf("%s %s", v.id, err.Error())
			}

			if !strings.Contains(e, "<type>other event</type>") {
				t.Errorf("%s expected event type <type>other event</type>", v.id)
			}
		}
	}
}

func TestToQuakeMLEvent(t *testing.T) {
	for _, input := range versions {
		b, err := ioutil.ReadFile("etc/" + input)
		if err != nil {
			t.Fatal(err)
		}

		var e string
		if e, err = toQuakeMLEvent(b); err != nil {
			t.Error(err)
		}

		if !strings.HasPrefix(e, `<event publicID="smi:nz.org.geonet/2015p768477">`) {
			t.Errorf("%s: quakeml fragment should start with <event...", input)
		}

		if !strings.HasSuffix(e, `</event>`) {
			t.Errorf("%s: quakeml fragment should end with </event>", input)
		}
	}

	var err error
	var f *os.File
	var b []byte

	// sc3ml 0.7
	if f, err = os.Open("etc/2015p768477_0.7.xml"); err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if b, err = ioutil.ReadAll(f); err != nil {
		t.Fatal(err)
	}

	b = bytes.Replace(b, sc3ml07, []byte(`<seiscomp xmlns="http://geofon.gfz-potsdam.de/ns/seiscomp3-schema/0.5" version="0.5">`), -1)

	if _, err := toQuakeMLEvent(b); err == nil {
		t.Error("expected error for version of sc3ml with no XSLT")
	}
}

func TestEventSave(t *testing.T) {
	setup(t)
	defer teardown()

	var err error
	var f *os.File
	var b []byte

	// sc3ml 0.7
	if f, err = os.Open("etc/2015p768477_0.7.xml"); err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if b, err = ioutil.ReadAll(f); err != nil {
		t.Fatal(err)
	}

	var e event

	if err = unmarshal(b, &e); err != nil {
		t.Error(err)
	}

	if _, err = db.Exec(`delete from fdsn.event where publicid = '2015p768477'`); err != nil {
		t.Error(err)
	}

	var count int

	if err = db.QueryRow(`select count(*) from fdsn.event where publicid = '2015p768477'`).Scan(&count); err != nil {
		t.Error(err)
	}

	if count != 0 {
		t.Error("found unpexected quake in the db")
	}

	if err = e.save(); err != nil {
		t.Error(err)
	}

	var mag float64

	if err = db.QueryRow(`select magnitude from fdsn.event where publicid = '2015p768477'`).Scan(&mag); err != nil {
		t.Error(err)
	}

	if mag != 5.691131913 {
		t.Error("mag should equal 5.691131913")
	}

	if err = e.save(); err != nil {
		t.Error("repeat save shouldn't error")
	}

	if err = db.QueryRow(`select magnitude from fdsn.event where publicid = '2015p768477'`).Scan(&mag); err != nil {
		t.Error(err)
	}

	if mag != 5.691131913 {
		t.Error("mag should equal 5.691131913")
	}

	e.ModificationTime, _ = time.Parse(time.RFC3339Nano, "2014-10-12T22:46:41.228824Z")
	e.Magnitude = 3.6

	if err = e.save(); err != nil {
		t.Error("save in past shouldn't update or error")
	}

	if err = db.QueryRow(`select magnitude from fdsn.event where publicid = '2015p768477'`).Scan(&mag); err != nil {
		t.Error(err)
	}

	if mag != 5.691131913 {
		t.Error("mag should equal 5.691131913 (no update for modification time earlier than in the db)")
	}

	e.ModificationTime, _ = time.Parse(time.RFC3339Nano, "2016-10-12T22:46:41.228824Z")
	e.Magnitude = 2.3
	if err = e.save(); err != nil {
		t.Error("update shouldn't error")
	}

	if err = db.QueryRow(`select magnitude from fdsn.event where publicid = '2015p768477'`).Scan(&mag); err != nil {
		t.Error(err)
	}

	if mag != 2.3 {
		t.Error("mag should equal 2.3 - modifcation time later than in the db")
	}
}

func setup(t *testing.T) {
	var err error

	db, err = sql.Open("postgres", "host=localhost connect_timeout=300 user=fdsn_w password=test dbname=fdsn sslmode=disable statement_timeout=600000")
	if err != nil {
		t.Fatalf("ERROR: problem with DB config: %s", err)
	}

	db.SetMaxIdleConns(1)
	db.SetMaxOpenConns(1)

	if err = db.Ping(); err != nil {
		t.Fatal("ERROR: problem pinging DB")
	}
}

func teardown() {
	db.Close()
}

func loc() string {
	_, _, l, _ := runtime.Caller(1)
	return "L" + strconv.Itoa(l)
}
