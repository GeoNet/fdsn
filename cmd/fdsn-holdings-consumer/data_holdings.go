package main

import (
	"github.com/GeoNet/fdsn/internal/holdings"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/lib/pq"
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
	r, err := saveHoldings.Exec(h.Network, h.Station, h.Channel, h.Location, h.Start, h.NumSamples, h.key, h.errorData, h.errorMsg)
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
	result, err := s3Client.GetObject(&s3.GetObjectInput{
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
