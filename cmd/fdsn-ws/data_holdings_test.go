package main

import (
	"github.com/GeoNet/fdsn/internal/fdsn"
	"github.com/GeoNet/fdsn/internal/holdings"
	"github.com/lib/pq"
	"testing"
	"time"
)

// http://www.postgresql.org/docs/9.4/static/errcodes-appendix.html
const (
	errorUniqueViolation pq.ErrorCode = "23505"
)

type holding struct {
	holdings.Holding
	key       string // the S3 bucket key
	errorData bool   // the miniSEED file has errors
	errorMsg  string // the cause of the errors
}

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

	d := fdsn.DataSearch{
		Network:  "NZ",
		Station:  "A.AZ",
		Location: "01",
		Channel:  "A.",
		Start:    start,
		End:      end,
	}

	keys, err := holdingsSearch(d)
	if err != nil {
		t.Error(err)
	}

	if len(keys) == 0 {
		t.Error("expected more than 0 keys")
	}
}

func (h *holding) save() error {
	r, err := h.saveHoldings()
	switch {
	case err != nil:
		return err
	case r == 1:
		return nil
	}

	_, err = h.saveStream()
	if err != nil {
		return err
	}

	_, err = h.saveHoldings()
	if err != nil {
		return err
	}

	return nil
}

func (h *holding) saveHoldings() (int64, error) {
	txn, err := db.Begin()

	_, err = txn.Exec(`DELETE FROM fdsn.holdings WHERE key=$1`, h.key)
	if err != nil {
		txn.Rollback()
		return 0, err
	}

	r, err := txn.Exec(`INSERT INTO fdsn.holdings (streamPK, start_time, numsamples, key, error_data, error_msg)
	SELECT streamPK, $5, $6, $7, $8, $9
	FROM fdsn.stream
	WHERE network = $1
	AND station = $2
	AND channel = $3
	AND location = $4`, h.Network, h.Station, h.Channel, h.Location, h.Start,
		h.NumSamples, h.key, h.errorData, h.errorMsg)
	if err != nil {
		txn.Rollback()
		return 0, err
	}

	err = txn.Commit()
	if err != nil {
		return 0, err
	}

	return r.RowsAffected()
}

func (h *holding) saveStream() (int64, error) {
	r, err := db.Exec(`INSERT INTO fdsn.stream (network, station, channel, location) VALUES($1, $2, $3, $4)`,
		h.Network, h.Station, h.Channel, h.Location)
	if err != nil {
		if u, ok := err.(*pq.Error); ok && u.Code == errorUniqueViolation {
			return 1, nil
		} else {
			return 0, err
		}
	}

	return r.RowsAffected()
}

func (h *holding) delete() error {
	_, err := db.Exec(`DELETE FROM fdsn.holdings WHERE key = $1`, h.key)
	return err
}
