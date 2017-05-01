package main

import (
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

func newS3Client(p client.ConfigProvider, cfgs ...*aws.Config) s3iface.S3API {
	return s3.New(p, cfgs...)
}

type s3DataSource struct {
	params       fdsnDataselectV1
	s3ClientFunc func(client.ConfigProvider, ...*aws.Config) s3iface.S3API
	bucket       string
	session      *session.Session
	cfg          *aws.Config
}

func newS3DataSource(bucket string, dsParams fdsnDataselectV1, maxRetries int) (*s3DataSource, error) {
	var err error
	var s3Session *session.Session

	cfg := aws.NewConfig()
	// either we specify maxRetries or a Retryer but no need for both, will use DefaultRetryer by default
	cfg.MaxRetries = &maxRetries
	// an interesting commit to fix timeouts/etc for strava: https://github.com/strava/goamz/commit/0013fca27c5c3b849d9b2d8c3c15f59a53ec38e9
	cfg.HTTPClient = http.DefaultClient
	cfg.HTTPClient.Timeout = HTTP_TIMEOUT

	if s3Session, err = session.NewSession(cfg); err != nil {
		return nil, err
	}

	s := s3DataSource{
		bucket:       bucket,
		params:       dsParams,
		s3ClientFunc: newS3Client, // Using a function so we can create new clients at will and mock with s3iface.
		session:      s3Session,
		cfg:          cfg,
	}

	return &s, nil
}

// getS3Object downloads the specified key from an S3 bucket and returns a byte slice containing the data.  This
// is thread safe and creates it's own S3 client.  This supports a context.Context to immediately cancel the download
// in case of aborts/timeouts/etc.
func (s *s3DataSource) getObject(ctx context.Context, key string) (b []byte, err error) {
	getParams := s3.GetObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	}

	// create a new client for each file to download, reuse the session and config
	c := s.s3ClientFunc(s.session)
	var output *s3.GetObjectOutput

	if output, err = c.GetObjectWithContext(ctx, &getParams); err != nil {
		return nil, err
	}
	defer output.Body.Close()

	if b, err = ioutil.ReadAll(output.Body); err != nil {
		return nil, err
	}

	return b, nil
}

// matchingKeys returns a slice of strings (S3 keys - filenames) that match the search parameters.
func (s *s3DataSource) matchingKeys(ctx context.Context) (keys []string, err error) {
	return holdingsSearch(s.regexp())
}

// regexp returns a regexp string that represents the search parameters.  It converts
// the '*', '?', ' ' and '--' characters to their regular expression equivalents for pattern matching with Go's regexp.
// It also handles multiple arguments, eg: two different networks.
func (s *s3DataSource) regexp() (network, station, location, channel string, start, end time.Time) {

	toPattern := func(params []string) (out string) {
		var newParams []string
		for _, param := range params {
			newParam := strings.Replace(param, `*`, `\w*`, -1)
			newParam = strings.Replace(newParam, `?`, `\w{1}`, -1)
			// blank or missing locations, we convert spaces and two dashes to wildcards for the regexp
			newParam = strings.Replace(newParam, `--`, `\w{2}`, -1)
			newParam = strings.Replace(newParam, ` `, `\w{1}`, -1)
			newParams = append(newParams, `(^`+newParam+`$)`)
		}

		return strings.Join(newParams, `|`)
	}

	return toPattern(s.params.Network),
		toPattern(s.params.Station),
		toPattern(s.params.Location),
		toPattern(s.params.Channel),
		s.params.StartTime.Time,
		s.params.EndTime.Time
}
