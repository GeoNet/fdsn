package main

import (
	"database/sql"
	"github.com/GeoNet/fdsn/internal/fdsn"
	"time"
)

type metric struct {
	Network, Station, Channel, Location string
	Key                                 string
	StartTime                           time.Time
	NumSamples                          int
	Error                               bool
	ErrorMessage                        string
}

// metricsSearch searches for data metrics matching the query.
// network, station, channel, and location are matched using POSIX regular expressions.
// https://www.postgresql.org/docs/9.3/static/functions-matching.html
// start and end should be set for all queries.  24 hours will be subtracted from the Start time and be added from the End time to include all records
// in each day long file.
func metricsSearch(d fdsn.DataSearch) ([]metric, error) {
	rows, err := db.Query(`WITH s AS (SELECT DISTINCT ON (network, station, channel, location) streamPK, network, station, channel, location
	FROM fdsn.stream WHERE network ~ $1
	AND station ~ $2
	AND channel ~ $3
	AND location ~ $4)
	SELECT DISTINCT ON (key) key, network, station, channel, location, start_time, numsamples, error_data, error_msg FROM s JOIN fdsn.holdings USING (streampk)
	WHERE start_time >= $5
	AND start_time <= $6`,
		d.Network, d.Station, d.Channel, d.Location, d.Start.Add(time.Hour*-24), d.End.Add(time.Hour*24))
	if err != nil {
		return []metric{}, err
	}

	var h []metric

	for rows.Next() {
		var v metric

		err = rows.Scan(&v.Key, &v.Network, &v.Station, &v.Channel, &v.Location, &v.StartTime, &v.NumSamples, &v.Error, &v.ErrorMessage)
		if err != nil {
			return []metric{}, err
		}
		h = append(h, v)
	}

	return h, nil
}

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
