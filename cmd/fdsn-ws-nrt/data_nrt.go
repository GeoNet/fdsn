package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/GeoNet/fdsn/internal/fdsn"
	"github.com/GeoNet/kit/metrics"
	"github.com/golang/groupcache"
	"log"
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
	defer func() {
		if err := timer.Track("holdingsSearchNrt"); err != nil {
			log.Println(err)
		}
	}()

	rows, err := db.Query(`WITH s AS (SELECT DISTINCT ON (streamPK) network, station, channel, location, streamPK
	FROM fdsn.stream WHERE network ~ $1
	AND station ~ $2
	AND channel ~ $3
	AND location ~ $4)
	SELECT network, station, channel, location, start_time FROM s JOIN fdsn.record USING (streamPK) WHERE start_time >= $5 AND start_time <= $6
	ORDER BY network, station, channel, location, start_time ASC`,
		d.Network, d.Station, d.Channel, d.Location, d.Start, d.End)
	if err != nil {
		return []string{}, err
	}
	defer rows.Close()

	var keys []string

	var n, s, c, l string
	var t time.Time

	for rows.Next() {
		err = rows.Scan(&n, &s, &c, &l, &t)
		if err != nil {
			return []string{}, err
		}
		keys = append(keys, fmt.Sprintf("%s_%s_%s_%s_%s", n, s, c, l, t.Format(time.RFC3339Nano)))
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
		err = recordCache.Get(context.TODO(), k, groupcache.AllocatingByteSliceSink(&rec))
		if err != nil && err != errNoData {
			return err
		}
	}

	return nil
}

// recordGetter implements groupcache.Getter for fetching miniSEED records from the cache.
// key is like "NZ_AWRB_HNN_23_2017-04-22T22:38:50.115Z"
// network_station_channel_location_time.RFC3339Nano
func recordGetter(ctx context.Context, key string, dest groupcache.Sink) error {
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

	return dest.SetBytes(b)
}
