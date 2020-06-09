package mseednrt_test

import (
	"bytes"
	"github.com/GeoNet/fdsn/internal/mseednrt"
	"github.com/GeoNet/fdsn/internal/mseednrt/fs"
	"runtime"
	"strconv"
	"testing"
	"time"
)

func TestCache_List(t *testing.T) {
	var results = []struct {
		id  string
		n   fs.NSLC
		exp int
	}{
		{
			id:  id(),
			n:   fs.NSLC{Network: "NZ", Station: "ABAZ", Location: "10", Channel: "EHE"},
			exp: 1,
		},
		{
			id:  id(),
			n:   fs.NSLC{Network: "NZ", Station: "ABAZ", Location: "10", Channel: ".*"},
			exp: 3,
		},
	}

	c := mseednrt.InitCache("TestCache_List", 1000000, 10000, time.Second*10, "fs/etc")

	for _, r := range results {
		l, err := c.List(r.n)
		if err != nil {
			t.Errorf("%s: %s\n", r.id, err)
		}

		if len(l) != r.exp {
			t.Errorf("%s: expected %d results got %d\n", r.id, r.exp, len(l))
		}
	}
}

func TestCache_Get(t *testing.T) {
	var results = []struct {
		id         string
		n          fs.NSLC
		start, end time.Time
		exp        int
	}{
		{
			id:    id(),
			n:     fs.NSLC{Network: "NZ", Station: "ABAZ", Location: "10", Channel: "EHE"},
			start: time.Date(2000, 1, 1, 1, 1, 1, 1, time.UTC),
			end:   time.Date(2020, 1, 1, 1, 1, 1, 1, time.UTC),
			exp:   512,
		},
		{
			id:    id(),
			n:     fs.NSLC{Network: "NZ", Station: "ABAZ", Location: "10", Channel: "EHE"},
			start: time.Date(2020, 1, 1, 1, 1, 1, 1, time.UTC),
			end:   time.Date(2021, 1, 1, 1, 1, 1, 1, time.UTC),
			exp:   0,
		},
		{
			id:    id(),
			n:     fs.NSLC{Network: "NZ", Station: "ABAZ", Location: "10", Channel: "EHE"},
			start: time.Date(1920, 1, 1, 1, 1, 1, 1, time.UTC),
			end:   time.Date(1921, 1, 1, 1, 1, 1, 1, time.UTC),
			exp:   0,
		},
	}

	c := mseednrt.InitCache("TestCache_Get", 1000000, 10000, time.Second*10, "fs/etc")

	var b bytes.Buffer

	for _, r := range results {
		b.Reset()

		n, err := c.Get(r.n, r.start, r.end, &b)
		if err != nil {
			t.Errorf("%s: %s", r.id, err)
		}

		if n != r.exp {
			t.Errorf("%s: expected to write %d bytes got %d\n", r.id, r.exp, n)
		}

		if b.Len() != r.exp {
			t.Errorf("%s: expected to %d bytes got %d\n", r.id, b.Len(), n)
		}
	}
}

func id() string {
	_, _, l, _ := runtime.Caller(1)
	return "L" + strconv.Itoa(l)
}
