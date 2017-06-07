package main

import (
	"github.com/lib/pq"
	"github.com/pkg/errors"
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
// slink can deliver duplicate packets and there may be multiple consumers
// this can cause races on updating the DB which are handled.
func (a *app) saveRecord(r record) error {
	n, err := a.saveRecordStmt.Exec(r.network, r.station, r.channel, r.location, r.start, r.raw, r.latency)
	if err != nil {
		if u, ok := err.(*pq.Error); ok && u.Code == errorUniqueViolation {
			// it is not an error if the record already exists.
			return nil
		} else {
			return err
		}
	}

	i, err := n.RowsAffected()
	if err != nil {
		return err
	}

	if i == 1 {
		// success - affected 1 row.  This should be the most common exit.
		return nil
	}

	// if no rows were affected - need to add the stream information
	_, err = a.db.Exec(`INSERT INTO fdsn.stream (network, station, channel, location) VALUES($1, $2, $3, $4)`,
		r.network, r.station, r.channel, r.location)
	if err != nil {
		if u, ok := err.(*pq.Error); ok && u.Code == errorUniqueViolation {
			// ignore unique errors, there is a DB race for multiple consumers adding stream information
		} else {
			return err
		}
	}

	// try to save the record again.
	n, err = a.saveRecordStmt.Exec(r.network, r.station, r.channel, r.location, r.start, r.raw, r.latency)
	if err != nil {
		if u, ok := err.(*pq.Error); ok && u.Code == errorUniqueViolation {
			return nil
		} else {
			return err
		}
	}

	i, err = n.RowsAffected()
	if err != nil {
		return err
	}

	if i == 1 {
		// success - affected 1 row.
		return nil
	}

	return errors.Errorf("affected zero rows saving record %s.%s.%s.%s", r.network, r.station, r.location, r.channel)
}
