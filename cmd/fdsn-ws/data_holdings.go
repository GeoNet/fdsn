package main

import (
	"database/sql"
	"time"
)

// holdingsSearch searches for S3 keys matching the query.
// network, station, channel, and location are matched using POSIX regular expressions.
// https://www.postgresql.org/docs/9.3/static/functions-matching.html
// start and end should be set for all queries.  24 hours will be subtracted from the Start time to include all records
// in each day long file.
func holdingsSearch(network, station, location, channel string, start, end time.Time) (keys []string, err error) {
	var rows *sql.Rows

	rows, err = db.Query(`WITH s AS (SELECT DISTINCT ON (network, station, channel, location) streamPK
	FROM fdsn.stream WHERE network ~ $1
	AND station ~ $2
	AND channel ~ $3
	AND location ~ $4)
	SELECT DISTINCT ON (key) key FROM s JOIN fdsn.holdings USING (streampk) WHERE start_time >= $5 AND start_time <= $6`,
		network, station, channel, location, start.Add(time.Hour*-24), end)
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
