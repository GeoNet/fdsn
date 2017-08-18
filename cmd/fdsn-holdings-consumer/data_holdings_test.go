package main

import (
	"database/sql"
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

	saveHoldings, err = db.Prepare(`INSERT INTO fdsn.holdings (streamPK, start_time, numsamples, key)
	SELECT streamPK, $5, $6, $7
	FROM fdsn.stream
	WHERE network = $1
	AND station = $2
	AND channel = $3
	AND location = $4
	ON CONFLICT (streamPK, key) DO UPDATE SET
	start_time = EXCLUDED.start_time,
	numsamples = EXCLUDED.numsamples`)
	if err != nil {
		t.Fatalf("preparing saveHoldings statement: %s", err.Error())
	}
}

func teardown() {
	db.Close()
}
