package main

import (
	"github.com/lib/pq"
	"time"
)

// http://www.postgresql.org/docs/9.4/static/errcodes-appendix.html
const (
	errorUniqueViolation pq.ErrorCode = "23505"
)

type record struct {
	network, station, channel, location string
	start                               time.Time
	latency                             float64
	raw                                 []byte
}

// save saves r to the DB adding the stream information to the DB if needed.
// slink can deliver duplicate packets and there may be multiple consumers.
func (r *record) save() error {
	// TODO - back off for DB connection errors.
	n, err := r.saveRecord()
	switch {
	case err != nil:
		return err
	case n == 1:
		return nil
	}

	_, err = r.saveStream()
	if err != nil {
		return err
	}

	_, err = r.saveRecord()
	if err != nil {
		return err
	}

	return nil
}

// saveRecord saves the record to the DB.
// Returns the number of rows affected and any errors.
// Saving a record that already exists in the DB is
// not an error and returns rows affected = 1.
func (r *record) saveRecord() (int64, error) {
	n, err := saveRecord.Exec(r.network, r.station, r.channel, r.location, r.start, r.raw, r.latency)
	if err != nil {
		if u, ok := err.(*pq.Error); ok && u.Code == errorUniqueViolation {
			return 1, nil
		} else {
			return 0, err
		}
	}

	return n.RowsAffected()
}

// saveStream saves the stream information in p to the DB.
// Returns the number of rows affected and any errors.
// Saving a stream that already exists in the DB is
// not an error and returns rows affected = 1.
func (r *record) saveStream() (int64, error) {
	n, err := db.Exec(`INSERT INTO fdsn.stream (network, station, channel, location) VALUES($1, $2, $3, $4)`,
		r.network, r.station, r.channel, r.location)
	if err != nil {
		if u, ok := err.(*pq.Error); ok && u.Code == errorUniqueViolation {
			return 1, nil
		} else {
			return 0, err
		}
	}

	return n.RowsAffected()
}
