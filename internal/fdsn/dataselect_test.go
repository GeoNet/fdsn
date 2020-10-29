package fdsn_test

import (
	"bytes"
	"fmt"
	"github.com/GeoNet/fdsn/internal/fdsn"
	"net/url"
	"reflect"
	"testing"
	"time"
)

func TestParsePostBody(t *testing.T) {
	// test the unmarshal method on dataSelectPostQuery that parses the POST body as per the FDSN spec.

	postBody := []byte(`quality=M
minimumlength=0.0
longestonly=FALSE
nodata=204
NZ ALRZ 10 EHN 2017-01-01T00:00:00 2017-01-10T00:00:00
NZ ABCD 10 E*? 2017-01-02T00:00:00 2017-01-03T00:00:00

`)

	t1, _ := time.Parse(time.RFC3339Nano, "2017-01-01T00:00:00.000000Z")
	t2, _ := time.Parse(time.RFC3339Nano, "2017-01-10T00:00:00.000000Z")
	t3, _ := time.Parse(time.RFC3339Nano, "2017-01-02T00:00:00.000000Z")
	t4, _ := time.Parse(time.RFC3339Nano, "2017-01-03T00:00:00.000000Z")

	var dsq []fdsn.DataSelect

	if err := fdsn.ParseDataSelectPost(bytes.NewReader(postBody), &dsq); err != nil {
		t.Fatal(err)
	}

	dsqExpected := []fdsn.DataSelect{
		{
			StartTime: fdsn.Time{Time: t1},
			EndTime:   fdsn.Time{Time: t2},
			Network:   []string{"NZ"},
			Station:   []string{"ALRZ"},
			Location:  []string{"10"},
			Channel:   []string{"EHN"},
			Format:    "miniseed",
			NoData:    204,
		},
		{
			StartTime: fdsn.Time{Time: t3},
			EndTime:   fdsn.Time{Time: t4},
			Network:   []string{"NZ"},
			Station:   []string{"ABCD"},
			Location:  []string{"10"},
			Channel:   []string{"E*?"},
			Format:    "miniseed",
			NoData:    204,
		},
	}

	if !reflect.DeepEqual(dsq, dsqExpected) {
		t.Errorf("structs do not match, expected: %v, observed: %v", dsqExpected, dsq)
	}
}

func TestParseGet(t *testing.T) {
	ts := "2020-01-01T00:00:00"
	te := "2020-01-01T01:00:00"
	u := url.Values{
		"network": []string{"NZ"},
		"station": []string{"ABAZ,AC*Z"},
		"channel": []string{"?HZ"},
		"start":   []string{ts},
		"end":     []string{te},
	}
	var dsq fdsn.DataSelect
	var err error
	if dsq, err = fdsn.ParseDataSelectGet(u); err != nil {
		t.Fatal(err)
	}

	var tms, tme fdsn.Time

	if err = tms.UnmarshalText([]byte(ts)); err != nil {
		t.Fatal(err)
	}
	if err = tme.UnmarshalText([]byte(te)); err != nil {
		t.Fatal(err)
	}
	dsqExpected := fdsn.DataSelect{
		StartTime: tms,
		EndTime:   tme,
		Network:   []string{"NZ"},
		Station:   []string{"ABAZ", "AC*Z"},
		Location:  []string{"*"},
		Channel:   []string{"?HZ"},
		Format:    "miniseed",
		NoData:    204,
	}

	if !reflect.DeepEqual(dsq, dsqExpected) {
		t.Errorf("structs do not match, expected: %+v, observed: %+v", dsqExpected, dsq)
	}

}

func TestGenRegex(t *testing.T) {
	// normal case
	r, err := fdsn.GenRegex([]string{"ABA0"}, false)
	if err != nil {
		t.Error(err)
	}
	if len(r) != 1 || r[0] != "^ABA0$" {
		t.Error(fmt.Sprintf("expect ^ABA0$ got %+v", r[0]))
	}

	// "--" empty location
	r, err = fdsn.GenRegex([]string{"--"}, true)
	if err != nil {
		t.Error(err)
	}
	if len(r) != 1 || r[0] != "^\\s{2}$" {
		t.Error(fmt.Sprintf("expect ^\\s{2}$ got %+v", r[0]))
	}

	// "?" and "*" special
	r, err = fdsn.GenRegex([]string{"A?Z*"}, false)
	if err != nil {
		t.Error(err)
	}
	if len(r) != 1 || r[0] != "^A.Z.*$" {
		t.Error(fmt.Sprintf("expect ^A.Z.*$ got %+v", r[0]))
	}

	// "--" (exactly 2 hyphens) means empty in FDSN
	_, err = fdsn.GenRegex([]string{"--"}, false)
	if err != nil {
		t.Error(fmt.Sprintf("expect to passed but rejected"))
	}

	_, err = fdsn.GenRegex([]string{"---"}, false)
	if err == nil {
		t.Error(fmt.Sprintf("expect to rejected but passed"))
	}

	// block all other chars, including valid regex since we're not supporting regex
	_, err = fdsn.GenRegex([]string{"*\\^{]"}, false)
	if err == nil {
		t.Error(fmt.Sprintf("expect to rejected but passed."))
	}

	_, err = fdsn.GenRegex([]string{"[E,H]H?"}, false)
	if err == nil {
		t.Error(fmt.Sprintf("expect to rejected but passed."))
	}

	_, err = fdsn.GenRegex([]string{"10,20"}, false)
	if err != nil {
		t.Error(fmt.Sprintf("expect to pass but rejected."))
	}
}

func TestWillBeEmpty(t *testing.T) {
	if shouldF := fdsn.WillBeEmpty("--"); shouldF != false {
		t.Error("expected to true got false")
	}
	if shouldT := fdsn.WillBeEmpty("^  $"); shouldT != true {
		t.Error("expected to true got false")
	}
	if shouldT := fdsn.WillBeEmpty("  "); shouldT != true {
		t.Error("expected to true got false")
	}
	if shouldF := fdsn.WillBeEmpty("^NZ$"); shouldF != false {
		t.Error("expected to false got true")
	}
	if shouldF := fdsn.WillBeEmpty("^WEL$"); shouldF != false {
		t.Error("expected to false got true")
	}
	if shouldF := fdsn.WillBeEmpty("^WEL$|^VIZ$"); shouldF != false {
		t.Error("expected to false got true")
	}
	if shouldT := fdsn.WillBeEmpty(",,"); shouldT != true {
		t.Error("expected to true got false")
	}
}
