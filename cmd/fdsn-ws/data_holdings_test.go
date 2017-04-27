package main

import (
	"github.com/GeoNet/fdsn/internal/holdings"
	"testing"
	"time"
)

func TestSaveHoldings(t *testing.T) {
	setup(t)
	defer teardown()

	h := holding{
		key: "NZ.ABAZ.01.ACE.D.2016.097",
		Holding: holdings.Holding{
			Network:    "NZ",
			Station:    "ABAZ",
			Channel:    "ACE",
			Location:   "01",
			Start:      time.Date(2016, time.January, 2, 0, 0, 0, 0, time.UTC),
			NumSamples: 500000,
		},
	}

	err := h.delete()

	var count int

	if err = db.QueryRow(`select count(*) from fdsn.holdings where key = 'NZ.ABAZ.01.ACE.D.2016.097'`).Scan(&count); err != nil {
		t.Error(err)
	}

	if count != 0 {
		t.Error("found unexpected holdings in the db")
	}

	err = h.save()
	if err != nil {
		t.Error(err)
	}

	if err = db.QueryRow(`select count(*) from fdsn.holdings where key = 'NZ.ABAZ.01.ACE.D.2016.097'`).Scan(&count); err != nil {
		t.Error(err)
	}

	if count != 1 {
		t.Error("expected holdings in the db")
	}

	// it is not an error to save the same key more than once.
	err = h.save()
	if err != nil {
		t.Error(err)
	}
}

func TestDataHoldingsSearch(t *testing.T) {
	setup(t)
	defer teardown()

	h := holding{
		key: "NZ.ABAZ.01.ACE.D.2016.097",
		Holding: holdings.Holding{
			Network:    "NZ",
			Station:    "ABAZ",
			Location:   "01",
			Channel:    "ACE",
			Start:      time.Date(2016, time.January, 2, 0, 0, 0, 0, time.UTC),
			NumSamples: 500000,
		},
	}

	err := h.save()
	if err != nil {
		t.Error(err)
	}

	start := time.Date(2016, time.January, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2017, time.January, 1, 0, 0, 0, 0, time.UTC)

	keys, err := holdingsSearch("NZ", "A.AZ", "01", "A.", start, end)
	if err != nil {
		t.Error(err)
	}

	if len(keys) == 0 {
		t.Error("expected more than 0 keys")
	}
}
