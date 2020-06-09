package mseedcache_test

import (
	"bytes"
	"fmt"
	"github.com/GeoNet/fdsn/internal/mseedcache"
	"github.com/GeoNet/kit/mseed"
	"io"
	"io/ioutil"
	"os"
	"runtime"
	"strconv"
	"testing"
	"time"
)

const recordLength int = 512

// getterFunc is an mseedcache.GetterFunc for testing.
// For S3 this would be implemented with GetObject.
func getterFunc(d mseedcache.DayFile) (io.ReadCloser, error) {
	name := fmt.Sprintf("etc/%s.%s.%s.%s.D.%d.%03d", d.Network, d.Station, d.Location, d.Channel, d.Date.Year(), d.Date.YearDay())

	return os.Open(name)
}

// modifiedFunc is an mseedcache.ModifiedFunc for testing.
// For S3 this would be implemented with HeadObject
func modifiedFunc(d mseedcache.DayFile) (time.Time, error) {
	name := fmt.Sprintf("etc/%s.%s.%s.%s.D.%d.%03d", d.Network, d.Station, d.Location, d.Channel, d.Date.Year(), d.Date.YearDay())

	inf, err := os.Stat(name)
	if err != nil {
		return time.Time{}, err
	}

	return inf.ModTime(), nil
}

// getRangeFunc is an mseedcache.GetRangeFunc for testing.
// For S3 this would be implemented with GetObject with a Range request.
func getRangeFunc(d mseedcache.DayFile, from, to int64) (io.ReadCloser, error) {
	name := fmt.Sprintf("etc/%s.%s.%s.%s.D.%d.%03d", d.Network, d.Station, d.Location, d.Channel, d.Date.Year(), d.Date.YearDay())

	in, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer in.Close()

	_, err = in.Seek(from, 0)
	if err != nil {
		return nil, err
	}

	var b bytes.Buffer

	_, err = io.CopyN(&b, in, to-from)
	if err != nil {
		return nil, err
	}

	return ioutil.NopCloser(&b), nil
}

// listerFunc is an mseedcache.ListerFunc for testing.
// For S3 this would be implemented with a ListObjectsV2 request.
func listerFunc(date time.Time) ([]mseedcache.NSLC, error) {
	return []mseedcache.NSLC{
		{Network: "NZ", Station: "ABAZ", Location: "10", Channel: "EHE"},
		{Network: "NZ", Station: "ABAZ", Location: "10", Channel: "EHN"},
	}, nil
}

func TestCacheList(t *testing.T) {
	var results = []struct {
		id         string
		start, end time.Time
		n          mseedcache.NSLC
		exp        []mseedcache.DayFile
	}{
		{
			id:    id(),
			n:     mseedcache.NSLC{Network: "NZ", Station: "ABAZ", Location: "10", Channel: "EHE"},
			start: time.Date(2016, 3, 17, 8, 0, 0, 0, time.UTC),
			end:   time.Date(2016, 3, 17, 9, 0, 0, 0, time.UTC),
			exp: []mseedcache.DayFile{
				{
					NSLC: mseedcache.NSLC{Network: "NZ", Station: "ABAZ", Location: "10", Channel: "EHE"},
					Date: time.Date(2016, 3, 17, 0, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			id:    id(),
			n:     mseedcache.NSLC{Network: "NZ", Station: "ABAZ", Location: "10", Channel: "EHE"},
			start: time.Date(2016, 3, 16, 8, 0, 0, 0, time.UTC),
			end:   time.Date(2016, 3, 17, 9, 0, 0, 0, time.UTC),
			exp: []mseedcache.DayFile{
				{
					NSLC: mseedcache.NSLC{Network: "NZ", Station: "ABAZ", Location: "10", Channel: "EHE"},
					Date: time.Date(2016, 3, 16, 0, 0, 0, 0, time.UTC),
				},
				{
					NSLC: mseedcache.NSLC{Network: "NZ", Station: "ABAZ", Location: "10", Channel: "EHE"},
					Date: time.Date(2016, 3, 17, 0, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			id:    id(),
			n:     mseedcache.NSLC{Network: "NZ", Station: "ABAZ", Location: "10", Channel: ".*"},
			start: time.Date(2016, 3, 17, 8, 0, 0, 0, time.UTC),
			end:   time.Date(2016, 3, 17, 9, 0, 0, 0, time.UTC),
			exp: []mseedcache.DayFile{
				{
					NSLC: mseedcache.NSLC{Network: "NZ", Station: "ABAZ", Location: "10", Channel: "EHE"},
					Date: time.Date(2016, 3, 17, 0, 0, 0, 0, time.UTC),
				},
				{
					NSLC: mseedcache.NSLC{Network: "NZ", Station: "ABAZ", Location: "10", Channel: "EHN"},
					Date: time.Date(2016, 3, 17, 0, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			id:    id(),
			n:     mseedcache.NSLC{Network: "NZ", Station: "ABAZ", Location: "10", Channel: "EHE"},
			start: time.Date(2016, 3, 17, 0, 0, 0, 0, time.UTC),
			end:   time.Date(2016, 3, 17, 23, 59, 59, 0, time.UTC),
			exp: []mseedcache.DayFile{
				{
					NSLC: mseedcache.NSLC{Network: "NZ", Station: "ABAZ", Location: "10", Channel: "EHE"},
					Date: time.Date(2016, 3, 16, 0, 0, 0, 0, time.UTC),
				},
				{
					NSLC: mseedcache.NSLC{Network: "NZ", Station: "ABAZ", Location: "10", Channel: "EHE"},
					Date: time.Date(2016, 3, 17, 0, 0, 0, 0, time.UTC),
				},
				{
					NSLC: mseedcache.NSLC{Network: "NZ", Station: "ABAZ", Location: "10", Channel: "EHE"},
					Date: time.Date(2016, 3, 18, 0, 0, 0, 0, time.UTC),
				},
			},
		},
	}

	c := mseedcache.InitCache("TestList", 10000000, 100000, time.Minute*30, getterFunc, listerFunc, modifiedFunc, getRangeFunc)

	for _, r := range results {
		d, err := c.List(r.n, r.start, r.end)
		if err != nil {
			t.Errorf("%s %s\n", r.id, err)
		}

		if len(r.exp) != len(d) {
			t.Errorf("%s expected %d results got %d\n", r.id, len(r.exp), len(d))
		}

		for i := 0; i < len(r.exp); i++ {
			if !d[i].Date.Truncate(time.Hour * 24).Equal(r.exp[i].Date.Truncate(time.Hour * 24)) {
				t.Errorf("%d truncated times don't match for entry %d expected %v got %v", r.id, i,
					r.exp[i].Date.Truncate(time.Hour*24),
					d[i].Date.Truncate(time.Hour*24))
			}
		}
	}
}

// TestCacheGet tests cache get using the test files in etc as the source.
func TestCacheGet(t *testing.T) {
	var results = []struct {
		id          string
		d           mseedcache.DayFile
		start, end  time.Time // start and end time are the time window query
		first, last time.Time // the first and last times of the returned data - usually slightly more than start and end
	}{
		// time window before the start and into first record (no file for the start of the query in the cache source)
		{
			id: id(),
			d: mseedcache.DayFile{
				NSLC: mseedcache.NSLC{Network: "NZ", Station: "ABAZ", Location: "10", Channel: "EHE"},
				Date: time.Date(2016, 3, 19, 0, 0, 0, 0, time.UTC),
			},
			start: time.Date(2016, 3, 17, 0, 0, 2, 0, time.UTC),
			end:   time.Date(2016, 3, 19, 0, 0, 3, 0, time.UTC),
			first: time.Date(2016, 3, 19, 0, 0, 1, 968393000, time.UTC),
			last:  time.Date(2016, 3, 19, 0, 0, 5, 928393000, time.UTC),
		},
		// time window in the first record
		{
			id: id(),
			d: mseedcache.DayFile{
				NSLC: mseedcache.NSLC{Network: "NZ", Station: "ABAZ", Location: "10", Channel: "EHE"},
				Date: time.Date(2016, 3, 19, 0, 0, 0, 0, time.UTC),
			},
			start: time.Date(2016, 3, 19, 0, 0, 2, 0, time.UTC),
			end:   time.Date(2016, 3, 19, 0, 0, 3, 0, time.UTC),
			first: time.Date(2016, 3, 19, 0, 0, 1, 968393000, time.UTC),
			last:  time.Date(2016, 3, 19, 0, 0, 5, 928393000, time.UTC),
		},
		// time window in the last record and past the end of the file
		{
			id: id(),
			d: mseedcache.DayFile{
				NSLC: mseedcache.NSLC{Network: "NZ", Station: "ABAZ", Location: "10", Channel: "EHE"},
				Date: time.Date(2016, 3, 19, 0, 0, 0, 0, time.UTC),
			},
			start: time.Date(2016, 3, 19, 23, 59, 59, 8391000, time.UTC),
			end:   time.Date(2016, 3, 21, 0, 0, 1, 0, time.UTC),
			first: time.Date(2016, 3, 19, 23, 59, 59, 8391000, time.UTC),
			last:  time.Date(2016, 3, 20, 0, 0, 2, 998391000, time.UTC),
		},
		//	time window in the last record
		{
			id: id(),
			d: mseedcache.DayFile{
				NSLC: mseedcache.NSLC{Network: "NZ", Station: "ABAZ", Location: "10", Channel: "EHE"},
				Date: time.Date(2016, 3, 19, 0, 0, 0, 0, time.UTC),
			},
			start: time.Date(2016, 3, 19, 23, 59, 59, 8391000, time.UTC),
			end:   time.Date(2016, 3, 20, 0, 0, 1, 0, time.UTC),
			first: time.Date(2016, 3, 19, 23, 59, 59, 8391000, time.UTC),
			last:  time.Date(2016, 3, 20, 0, 0, 2, 998391000, time.UTC),
		},
		// second to last record
		{
			id: id(),
			d: mseedcache.DayFile{
				NSLC: mseedcache.NSLC{Network: "NZ", Station: "ABAZ", Location: "10", Channel: "EHE"},
				Date: time.Date(2016, 3, 19, 0, 0, 0, 0, time.UTC),
			},
			start: time.Date(2016, 3, 19, 23, 59, 55, 0, time.UTC),
			end:   time.Date(2016, 3, 19, 23, 59, 57, 0, time.UTC),
			first: time.Date(2016, 3, 19, 23, 59, 54, 908391000, time.UTC),
			last:  time.Date(2016, 3, 19, 23, 59, 58, 998391000, time.UTC),
		},
		// multiple records inside the file
		{
			id: id(),
			d: mseedcache.DayFile{
				NSLC: mseedcache.NSLC{Network: "NZ", Station: "ABAZ", Location: "10", Channel: "EHE"},
				Date: time.Date(2016, 3, 19, 0, 0, 0, 0, time.UTC),
			},
			start: time.Date(2016, 3, 19, 23, 59, 22, 0, time.UTC),
			end:   time.Date(2016, 3, 19, 23, 59, 57, 0, time.UTC),
			first: time.Date(2016, 3, 19, 23, 59, 21, 848391000, time.UTC),
			last:  time.Date(2016, 3, 19, 23, 59, 58, 998391000, time.UTC),
		},
		// the whole file, time window inside the file
		{
			id: id(),
			d: mseedcache.DayFile{
				NSLC: mseedcache.NSLC{Network: "NZ", Station: "ABAZ", Location: "10", Channel: "EHE"},
				Date: time.Date(2016, 3, 19, 0, 0, 0, 0, time.UTC),
			},
			start: time.Date(2016, 3, 19, 0, 0, 0, 0, time.UTC),
			end:   time.Date(2016, 3, 19, 23, 59, 59, 0, time.UTC),
			first: time.Date(2016, 3, 19, 0, 0, 1, 968393000, time.UTC),
			last:  time.Date(2016, 3, 20, 0, 0, 2, 998391000, time.UTC),
		},
		// the whole file, time window outside the file
		{
			id: id(),
			d: mseedcache.DayFile{
				NSLC: mseedcache.NSLC{Network: "NZ", Station: "ABAZ", Location: "10", Channel: "EHE"},
				Date: time.Date(2016, 3, 19, 0, 0, 0, 0, time.UTC),
			},
			start: time.Date(2016, 3, 18, 0, 0, 0, 0, time.UTC),
			end:   time.Date(2016, 3, 20, 23, 59, 59, 0, time.UTC),
			first: time.Date(2016, 3, 19, 0, 0, 1, 968393000, time.UTC),
			last:  time.Date(2016, 3, 20, 0, 0, 2, 998391000, time.UTC),
		},
	}

	c := mseedcache.InitCache("TestIndex", 10000000, 1000000, time.Minute*30, getterFunc, listerFunc, modifiedFunc, getRangeFunc)

	msr := mseed.NewMSRecord()
	defer mseed.FreeMSRecord(msr)

	record := make([]byte, recordLength)

	for _, r := range results {
		var b bytes.Buffer

		_, err := c.Get(r.d, r.start, r.end, &b)

		if err != nil {
			t.Error(err)
		}

		var f, l time.Time

	loop:
		for {
			_, err = b.Read(record)
			switch {
			case err == io.EOF:
				break loop
			case err != nil:
				t.Error(err)
			}

			err = msr.Unpack(record, recordLength, 1, 0)
			if err != nil {
				t.Error(err)
			}

			if f.IsZero() {
				f = msr.Starttime()
			}
		}

		l = msr.Endtime()

		if !r.first.Equal(f) {
			t.Errorf("%s expected first time %s got %s\n", r.id, r.first, f)
		}

		if !r.last.Equal(l) {
			t.Errorf("%s expected last time %s got %s\n", r.id, r.last, l)
		}
	}
}

// BenchmarkCacheMSeed uses the cache to filter mSEED.  Shows the performance increase from
// using a cached index file for the mSEED.
//
// goos: linux
// goarch: amd64
// pkg: github.com/GeoNet/fdsn-ng/internal/mseedcache
// 300	   5018508 ns/op
func BenchmarkCacheMSeed(b *testing.B) {
	c := mseedcache.InitCache("BenchmarkCacheMSeed", 10000000, 1000000, time.Minute*30, getterFunc, listerFunc, modifiedFunc, getRangeFunc)

	d := mseedcache.DayFile{
		NSLC: mseedcache.NSLC{Network: "NZ", Station: "ABAZ", Location: "10", Channel: "EHE"},
		Date: time.Date(2016, 3, 19, 0, 0, 0, 0, time.UTC),
	}
	start := time.Date(2016, 3, 19, 23, 59, 22, 0, time.UTC)
	end := time.Date(2016, 3, 19, 23, 59, 57, 0, time.UTC)

	b.Run("Get mSEED from cache", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			_, err := c.Get(d, start, end, ioutil.Discard)
			if err != nil {
				b.Error(err)
			}
		}
	})
}

// BenchmarkFilterMseed filters records from an mSEED file directly (reads all record headers every time).
//
// goos: linux
// goarch: amd64
// pkg: github.com/GeoNet/fdsn-ng/internal/mseedcache
// 10	 106533934 ns/op
// PASS
//
// Process finished with exit code 0
func BenchmarkFilterMseed(b *testing.B) {
	msr := mseed.NewMSRecord()
	defer mseed.FreeMSRecord(msr)

	record := make([]byte, recordLength)

	start := time.Date(2016, 3, 19, 23, 59, 22, 0, time.UTC)
	end := time.Date(2016, 3, 19, 23, 59, 57, 0, time.UTC)

	for n := 0; n < b.N; n++ {
		f, err := os.Open("etc/NZ.ABAZ.10.EHE.D.2016.079")
		if err != nil {
			b.Fatal(err)
		}

	loop:
		for {
			_, err = io.ReadFull(f, record)
			switch {
			case err == io.EOF:
				break loop
			case err != nil:
				b.Error(err)
			}

			err = msr.Unpack(record, recordLength, 1, 0)
			if err != nil {
				b.Error(err)
			}

			if msr.Starttime().After(start) && msr.Endtime().Before(end) {
				_, err = ioutil.Discard.Write(record)
				if err != nil {
					b.Error(err)
				}
			}
		}

		err = f.Close()
		if err != nil {
			b.Error(err)
		}
	}
}

func id() string {
	_, _, l, _ := runtime.Caller(1)
	return "L" + strconv.Itoa(l)
}
