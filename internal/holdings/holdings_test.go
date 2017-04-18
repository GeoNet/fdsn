package holdings_test

import (
	"github.com/GeoNet/fdsn/internal/holdings"
	"os"
	"reflect"
	"testing"
	"time"
)

type result struct {
	file string
	h    holdings.Holding
}

var results = []result{
	{
		file: "etc/NZ.ABAZ.10.EHE.D.2016.079",
		h: holdings.Holding{
			Network: "NZ", Station: "ABAZ", Channel: "EHE", Location: "10",
			Start:      time.Date(2016, time.March, 19, 0, 0, 1, 968393*1000, time.UTC),
			NumSamples: 8640104,
		},
	},
	{
		file: "etc/NZ.ABAZ..LOG.D.2016.186",
		h: holdings.Holding{
			Network: "NZ", Station: "ABAZ", Channel: "LOG", Location: "",
			Start:      time.Date(2016, time.July, 4, 23, 57, 14, 3984*100000, time.UTC),
			NumSamples: 375,
		},
	},
}

func TestHoldings(t *testing.T) {
	for _, e := range results {
		r, err := os.Open(e.file)
		if err != nil {
			t.Fatalf("%s %s", e.file, err)
		}

		h, err := holdings.SingleStream(r)
		r.Close()
		if err != nil {
			t.Errorf("%s %s", e.file, err)
		}

		if !reflect.DeepEqual(e.h, h) {
			t.Errorf("%s holdings results not equal expected %+v got %+v", e.file, e.h, h)
		}
	}
}
