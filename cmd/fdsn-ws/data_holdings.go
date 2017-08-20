package main

import (
	"database/sql"
	"github.com/GeoNet/fdsn/internal/fdsn"
	"time"
)

// holdingsSearch searches for S3 keys matching the query.
// network, station, channel, and location are matched using POSIX regular expressions.
// https://www.postgresql.org/docs/9.3/static/functions-matching.html
// start and end should be set for all queries.  24 hours will be subtracted from the Start time and be added from the End time to include all records
// in each day long file.
func holdingsSearch(d fdsn.DataSearch) (keys []string, err error) {
	var rows *sql.Rows

	rows, err = db.Query(`WITH s AS (SELECT DISTINCT ON (network, station, channel, location) streamPK
	FROM fdsn.stream WHERE network ~ $1
	AND station ~ $2
	AND channel ~ $3
	AND location ~ $4)
	SELECT DISTINCT ON (key) key FROM s JOIN fdsn.holdings USING (streampk)
	WHERE start_time >= $5
	AND start_time <= $6
	AND error_data = false`,
		d.Network, d.Station, d.Channel, d.Location, d.Start.Add(time.Hour*-24), d.End.Add(time.Hour*24))
	if err != nil {
		return
	}

	var s string

	for rows.Next() {
		err = rows.Scan(&s)
		if err != nil {
			return
		}
		keys = append(keys, s)
	}

	return
}
