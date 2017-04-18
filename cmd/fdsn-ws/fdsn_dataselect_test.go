package main

import (
	"bytes"
	"context"
	"reflect"
	"testing"
	"time"
)

func TestParsePostBody(t *testing.T) {
	// test the unmarshal method on dataSelectPostQuery that parses the POST body as per the FDSN spec.

	postBody := []byte(`quality=M
minimumlength=0.0
longestonly=FALSE
NZ ALRZ 10 EHN 2017-01-01T00:00:00 2017-01-10T00:00:00
NZ ABCD 10 E*? 2017-01-02T00:00:00 2017-01-03T00:00:00

`)

	t1, _ := time.Parse(time.RFC3339Nano, "2017-01-01T00:00:00.000000Z")
	t2, _ := time.Parse(time.RFC3339Nano, "2017-01-10T00:00:00.000000Z")
	t3, _ := time.Parse(time.RFC3339Nano, "2017-01-02T00:00:00.000000Z")
	t4, _ := time.Parse(time.RFC3339Nano, "2017-01-03T00:00:00.000000Z")

	var dsq dataSelectPostQuery
	if err := dsq.unmarshal(postBody); err != nil {
		t.Fatal(err)
	}

	dsqExpected := []fdsnDataselectV1{
		{
			StartTime: Time{t1},
			EndTime:   Time{t2},
			Network:   []string{"NZ"},
			Station:   []string{"ALRZ"},
			Location:  []string{"10"},
			Channel:   []string{"EHN"},
		},
		{
			StartTime: Time{t3},
			EndTime:   Time{t4},
			Network:   []string{"NZ"},
			Station:   []string{"ABCD"},
			Location:  []string{"10"},
			Channel:   []string{"E*?"},
		},
	}

	if !reflect.DeepEqual(dsq, dataSelectPostQuery(dsqExpected)) {
		t.Errorf("structs do not match, expected: %v, observed: %v", dsqExpected, dsq)
	}
}

// Benchmark for parsing miniseed data.  When benchmarking do not use the -race compile flag.
// CPU profiling: `go test -bench=. -cpuprofile=fetchfile.prof`, analyse with cmd `go tool pprof fetchfile.prof`
// Results show 419 ns/op for parsing miniseed headers, which is orders of magnitude small than the download time.
func BenchmarkFetchFile(b *testing.B) {
	// Download a miniseed file and store as an in-memory ReadCloser for use with the benchmark
	var err error
	params := fdsnDataselectV1{Network: []string{"NZ"}, Station: []string{"ALRZ"}, Location: []string{"10"}, Channel: []string{"EHN"}}
	params.StartTime.Time, err = time.Parse(time.RFC3339Nano, "2017-01-08T00:00:00.000000000Z")
	if err != nil {
		b.Fatal(err)
	}

	params.EndTime.Time, err = time.Parse(time.RFC3339Nano, "2075-01-10T00:00:00.000000000Z")
	if err != nil {
		b.Fatal(err)
	}

	ds, err := newS3DataSource(S3_BUCKET, params, MAX_RETRIES)
	var matchingKeys []string
	matchingKeys, err = ds.matchingKeys(context.Background())
	if err != nil {
		b.Fatal(err)
	}

	if len(matchingKeys) == 0 {
		b.Fatal("no matching files")
	}

	var matchData []byte
	if matchData, err = ds.getObject(context.Background(), matchingKeys[0]); err != nil {
		b.Fatal(err)
	}

	buff := bytes.NewBuffer(matchData)

	benchmarks := []struct {
		name string
		ds   *s3DataSource
	}{
		{"NZ.ALRZ.10.EHN.D.2017.008", ds},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				m := match{dataSource: bm.ds, key: matchingKeys[0]}

				// check to see that we can parse the data, also worth profiling
				if _, err := m.parse(buff); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
