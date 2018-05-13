package main

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/GeoNet/fdsn/internal/fdsn"
	"github.com/GeoNet/kit/metrics"
	"github.com/golang/groupcache"
	"strings"
	"time"
)

var errNoData = errors.New("no data")

// holdingsSearchNrt searches for near real time records matching the query.
// network, station, channel, and location are matched using POSIX regular expressions.
// https://www.postgresql.org/docs/9.3/static/functions-matching.html
// start and end should be set for all queries.
func holdingsSearchNrt(d fdsn.DataSearch) ([]string, error) {
	timer := metrics.Start()
	defer timer.Track("holdingsSearchNrt")

	// With each record is about 10s long, we query for records for 1 minute prior to start_time and 1 minute after end_time, then filter them afterwards.
	rows, err := db.Query(`WITH s AS (SELECT DISTINCT ON (streamPK) network, station, channel, location, streamPK
	FROM fdsn.stream WHERE network ~ $1
	AND station ~ $2
	AND channel ~ $3
	AND location ~ $4)
	SELECT network, station, channel, location, start_time FROM s JOIN fdsn.record USING (streamPK) WHERE start_time >= $5 AND start_time <= $6
	ORDER BY network, station, channel, location, start_time ASC`,
		d.Network, d.Station, d.Channel, d.Location, d.Start.Add(time.Minute*-1), d.End.Add(time.Minute*1))
	if err != nil {
		return []string{}, err
	}
	defer rows.Close()

	var keys []string

	var n, s, c, l string
	var t time.Time

	crossedStart := false
	prevKey := ""
	for rows.Next() {
		err = rows.Scan(&n, &s, &c, &l, &t)
		if err != nil {
			return []string{}, err
		}

		if t.After(d.End) {
			break
		}

		key := fmt.Sprintf("%s_%s_%s_%s_%s", n, s, c, l, t.Format(time.RFC3339Nano))
		if !crossedStart {
			if t.After(d.Start) {
				// Previous record is the record which crossed (or on) the start time
				crossedStart = true
				if prevKey != "" {
					keys = append(keys, prevKey)
				}
				keys = append(keys, key)
			} else {
				prevKey = key
			}
		} else {
			keys = append(keys, key)
		}
	}

	return keys, nil
}

// primeCache fills the miniSEED record cache from the DB.  It fills for records
// more recent than start.
func primeCache(start time.Time) error {
	rows, err := db.Query(`WITH r AS (SELECT streamPk, start_time
				FROM fdsn.record WHERE start_time > $1)
				SELECT network, station, channel, location, start_time FROM fdsn.stream JOIN r USING (streamPK)
				ORDER BY start_time DESC`, start)
	if err != nil {
		return err
	}
	defer rows.Close()

	var keys []string

	var n, s, c, l string
	var t time.Time

	for rows.Next() {
		err = rows.Scan(&n, &s, &c, &l, &t)
		if err != nil {
			return err
		}
		keys = append(keys, fmt.Sprintf("%s_%s_%s_%s_%s", n, s, c, l, t.Format(time.RFC3339Nano)))
	}

	rows.Close()

	var rec []byte

	for _, k := range keys {
		err = recordCache.Get(nil, k, groupcache.AllocatingByteSliceSink(&rec))
		if err != nil && err != errNoData {
			return err
		}
	}

	return nil
}

// recordGetter implements groupcache.Getter for fetching miniSEED records from the cache.
// key is like "NZ_AWRB_HNN_23_2017-04-22T22:38:50.115Z"
// network_station_channel_location_time.RFC3339Nano
func recordGetter(ctx groupcache.Context, key string, dest groupcache.Sink) error {
	p := strings.Split(key, "_")
	if len(p) != 5 {
		return errors.New("expected 5 parts to key: " + key)
	}

	t, err := time.Parse(time.RFC3339Nano, p[4])
	if err != nil {
		return err
	}

	var b []byte
	err = recordStmt.QueryRow(p[0], p[1], p[2], p[3], t).Scan(&b)
	if err != nil {
		if err == sql.ErrNoRows {
			return errNoData
		}
		return err
	}

	dest.SetBytes(b)
	return nil
}
