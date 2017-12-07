package fdsn_test

import (
	"bytes"
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
