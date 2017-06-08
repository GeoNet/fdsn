package main

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/GeoNet/fdsn/internal/fdsn"
	"github.com/golang/groupcache"
	"strings"
	"time"
)

var errNoData = errors.New("no data")

var recordCache = groupcache.NewGroup("record", recordCacheSize, groupcache.GetterFunc(
	func(ctx groupcache.Context, key string, dest groupcache.Sink) error {

		// key is like "NZ_AWRB_HNN_23_2017-04-22T22:38:50.115Z"
		// network_station_channel_location_time.RFC3339Nano

		p := strings.Split(key, "_")
		if len(p) != 5 {
			return errors.New("expected 5 parts to key: " + key)
		}

		t, err := time.Parse(time.RFC3339Nano, p[4])
		if err != nil {
			return err
		}

		var b []byte
		err = db.QueryRow(`SELECT raw FROM fdsn.record WHERE streampk =
                                  (SELECT streampk FROM fdsn.stream WHERE network = $1 AND station = $2 AND channel = $3 AND location = $4)
	                          AND start_time = $5`, p[0], p[1], p[2], p[3], t).Scan(&b)
		if err != nil {
			if err == sql.ErrNoRows {
				return errNoData
			}
			return err
		}

		dest.SetBytes(b)
		return nil
	},
))

// holdingsSearchNrt searches for near real time records matching the query.
// network, station, channel, and location are matched using POSIX regular expressions.
// https://www.postgresql.org/docs/9.3/static/functions-matching.html
// start and end should be set for all queries.
func holdingsSearchNrt(d fdsn.DataSearch) ([]string, error) {
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
