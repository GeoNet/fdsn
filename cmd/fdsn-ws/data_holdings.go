package main

import (
	"bytes"
	"database/sql"
	"github.com/GeoNet/fdsn/internal/holdings"
	"github.com/GeoNet/weft"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/lib/pq"
	"log"
	"net/http"
	"strings"
	"time"
)

// http://www.postgresql.org/docs/9.4/static/errcodes-appendix.html
const (
	errorUniqueViolation pq.ErrorCode = "23505"
)

type holding struct {
	holdings.Holding
	key string // the S3 bucket key
}

// create an S3 session and client just for holdings.
// TODO - possibly refactor this to use a shared session etc.
var holdingsSession *session.Session
var holdingsClient *s3.S3

func init() {
	holdingsSession, err := session.NewSession()
	if err != nil {
		log.Print(err)
	}

	holdingsSession.Config.Retryer = client.DefaultRetryer{NumMaxRetries: 3}
	holdingsClient = s3.New(holdingsSession)
}

// handles PUT requests for URLs with key files e.g., /holdings/NZ.ABAZ.01.ACE.D.2016.097
func holdingsHandler(r *http.Request, h http.Header, b *bytes.Buffer) *weft.Result {
	switch r.Method {
	case "PUT":
		_, p, ok := r.BasicAuth()
		if !ok || p == "" || p != key {
			return &weft.Unauthorized
		}

		if res := weft.CheckQuery(r, []string{}, []string{}); !res.Ok {
			return res
		}

		hld, err := holdingS3(S3_BUCKET, strings.TrimPrefix(r.URL.Path, "/holdings/"))
		if err != nil {
			log.Printf("getting holdings for %s: %s", strings.TrimPrefix(r.URL.Path, "/holdings/"), err)
			return &weft.NotFound
		}

		err = hld.save()
		if err != nil {
			return weft.InternalServerError(err)
		}

		return &weft.StatusOK
	case "DELETE":
		_, p, ok := r.BasicAuth()
		if !ok || p == "" || p != key {
			return &weft.Unauthorized
		}

		if res := weft.CheckQuery(r, []string{}, []string{}); !res.Ok {
			return res
		}

		h := holding{key: strings.TrimPrefix(r.URL.Path, "/holdings/")}

		err := h.delete()
		if err != nil {
			return weft.InternalServerError(err)
		}

		return &weft.StatusOK
	default:
		return &weft.MethodNotAllowed
	}
}

// holdingsSearch searches for S3 keys matching the query.
// network, station, channel, and location can contain the LIKE matching
// postgres operators _ (any single character) or % (any sequence of zero or more characters).
// https://www.postgresql.org/docs/9.3/static/functions-matching.html
// start and end should be set for all queries.
func holdingsSearch(network, station, channel, location string, start, end time.Time) (keys []string, err error) {
	var rows *sql.Rows

	rows, err = db.Query(`WITH s AS (SELECT DISTINCT ON (network, station, channel, location) streamPK
	FROM fdsn.stream WHERE network LIKE $1
	AND station LIKE $2
	AND channel LIKE $3
	AND location LIKE $4)
	SELECT DISTINCT ON (key) key FROM s JOIN fdsn.holdings USING (streampk) WHERE start_time >= $5 AND start_time <= $6`,
		network, station, channel, location, start, end)
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

	r, err := txn.Exec(`INSERT INTO fdsn.holdings (streamPK, start_time, numsamples, key)
	SELECT streamPK, $5, $6, $7
	FROM fdsn.stream
	WHERE network = $1
	AND station = $2
	AND channel = $3
	AND location = $4`, h.Network, h.Station, h.Channel, h.Location, h.Start,
		h.NumSamples, h.key)
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

func holdingS3(bucket, key string) (holding, error) {
	result, err := holdingsClient.GetObject(&s3.GetObjectInput{
		Key:    aws.String(key),
		Bucket: aws.String(bucket),
	})

	if err != nil {
		return holding{}, err
	}
	defer result.Body.Close()

	h, err := holdings.SingleStream(result.Body)
	if err != nil {
		return holding{}, err
	}

	return holding{key: key, Holding: h}, nil
}
