package fdsn_test

import (
	"bytes"
	"fmt"
	"github.com/GeoNet/fdsn/internal/fdsn"
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
}
