package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/GeoNet/fdsn/internal/fdsn"
	"github.com/GeoNet/kit/mseed"
	"github.com/GeoNet/kit/weft"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"reflect"
	"strings"
	"text/template"
	"time"
)

const (
	// miniSEED record length
	RECORDLEN int = 512
	// the maximum number of queries in a POST request
	MAX_QUERIES int = 60
	// Limit the number of input files (each file is max ~10 MB).
	MAX_FILES int = 60
)

var (
	s3Session              *session.Session
	s3Client               *s3.S3
	fdsnDataselectWadlFile []byte
	fdsnDataselectIndex    []byte
)

type dataSelect struct {
	d    fdsn.DataSearch
	keys []string
}

func init() {
	// Handle comma separated parameters (eg: net, sta, loc, cha, etc)
	decoder.RegisterConverter([]string{}, func(input string) reflect.Value {
		return reflect.ValueOf(strings.Split(input, ","))
	})

	var err error
	var b bytes.Buffer

	t, err := template.New("t").ParseFiles("assets/tmpl/fdsn-ws-dataselect.wadl")
	if err != nil {
		log.Printf("error parsing assets/tmpl/fdsn-ws-dataselect.wadl: %s", err.Error())
	}
	err = t.ExecuteTemplate(&b, "body", os.Getenv("HOST_CNAME"))
	if err != nil {
		log.Printf("error executing assets/tmpl/fdsn-ws-dataselect.wadl: %s", err.Error())
	}
	fdsnDataselectWadlFile = b.Bytes()

	fdsnDataselectIndex, err = ioutil.ReadFile("assets/fdsn-ws-dataselect.html")
	if err != nil {
		log.Printf("error reading assets/fdsn-ws-dataselect.html: %s", err.Error())
	}

	s3Session, err = session.NewSession()
	if err != nil {
		log.Fatalf("creating S3 session: %s", err)
	}

	s3Session.Config.Retryer = client.DefaultRetryer{NumMaxRetries: 3}
	s3Client = s3.New(s3Session)
}

// fdsnDataMetricsV1Handler handles all datametrics queries.
func fdsnDataMetricsV1Handler(r *http.Request, h http.Header, b *bytes.Buffer) error {
	var params []fdsn.DataSelect

	switch r.Method {
	case "POST":
		defer r.Body.Close()
		if err := fdsn.ParseDataSelectPost(r.Body, &params); err != nil {
			return weft.StatusError{Code: http.StatusBadRequest, Err: err}
		}
	case "GET":
		d, err := fdsn.ParseDataSelectGet(r.URL.Query())
		if err != nil {
			return weft.StatusError{Code: http.StatusBadRequest, Err: err}
		}

		params = append(params, d)
	default:
		return weft.StatusError{Code: http.StatusMethodNotAllowed}
	}

	if len(params) > MAX_QUERIES {
		return weft.StatusError{Code: http.StatusRequestEntityTooLarge,
			Err: fmt.Errorf("number of queries in the POST request: %d exceeded the limit: %d", len(params), MAX_QUERIES)}
	}

	var metrics []metric

	for _, v := range params {
		d, err := v.Regexp()
		if err != nil {
			return err
		}
		m, err := metricsSearch(d)
		if err != nil {
			return err
		}

		if len(metrics) > MAX_FILES {
			return weft.StatusError{Code: http.StatusRequestEntityTooLarge,
				Err: fmt.Errorf("number of queries in the POST request: %d exceeded the limit: %d", len(params), MAX_QUERIES)}
		}

		metrics = append(metrics, m...)
	}

	if len(metrics) == 0 {
		return weft.StatusError{Code: params[0].NoData, Err: fmt.Errorf("%s", "no results for specified query")}
	}

	by, err := json.Marshal(metrics)
	if err != nil {
		return err
	}

	b.Write(by)
	h.Set("Content-Type", "application/json")

	return nil
}

// fdsnDataselectV1Handler handles all dataselect queries.  It searches for matching keys in S3 and
// fetches them, writing matching records to w in the same order they were requested.
// Results are streamed to the client so a 200 can still be followed by errors which will not
// be reported to the client.  The potentially large response sizes make this the simplest solution.
func fdsnDataselectV1Handler(r *http.Request, w http.ResponseWriter) (int64, error) {
	var params []fdsn.DataSelect

	switch r.Method {
	case "POST":
		defer r.Body.Close()
		if err := fdsn.ParseDataSelectPost(r.Body, &params); err != nil {
			return 0, weft.StatusError{Code: http.StatusBadRequest, Err: err}
		}
		if len(params) == 0 {
			return 0, weft.StatusError{Code: NO_DATA, Err: fmt.Errorf("%s", "unable to parse post request")}
		}
	case "GET":
		d, err := fdsn.ParseDataSelectGet(r.URL.Query())
		if err != nil {
			return 0, weft.StatusError{Code: http.StatusBadRequest, Err: err}
		}

		params = append(params, d)
	default:
		return 0, weft.StatusError{Code: http.StatusMethodNotAllowed}
	}

	if len(params) > MAX_QUERIES {
		return 0, weft.StatusError{Code: http.StatusRequestEntityTooLarge,
			Err: fmt.Errorf("number of queries in the POST request: %d exceeded the limit: %d", len(params), MAX_QUERIES)}
	}

	// search the holdings DB for the files to fetch from S3.
	// return an error if this would be to many files.
	var request []dataSelect
	var files int

	gtHalfHour := false //tracks whether any requests are for data longer than 30mins

	for _, v := range params {
		//flick gtHalfHour to true if the request is longer than half an hour
		gtHalfHour = gtHalfHour || v.EndTime.Sub(time.Time(v.StartTime.Time)) > time.Minute*30

		d, err := v.Regexp()
		if err != nil {
			return 0, weft.StatusError{Code: http.StatusBadRequest, Err: err}
		}
		keys, err := holdingsSearch(d)
		if err != nil {
			return 0, err
		}

		files += len(keys)

		if files > MAX_FILES && gtHalfHour {
			return 0, weft.StatusError{Code: http.StatusRequestEntityTooLarge,
				Err: fmt.Errorf("number of queries in the POST request: %d exceeded the limit: %d", len(params), MAX_QUERIES)}
		}

		request = append(request, dataSelect{d: d, keys: keys})
	}

	if files == 0 {
		return 0, weft.StatusError{Code: params[0].NoData, Err: fmt.Errorf("%s", "no results for specified query")}
	}

	// Fetch the miniSEED files from S3.  Parse them and write
	// the records inside the time window for the query to the client.
	msr := mseed.NewMSRecord()
	defer mseed.FreeMSRecord(msr)

	record := make([]byte, RECORDLEN)

	w.Header().Set("Content-Type", "application/vnd.fdsn.mseed")

	var n int
	var written int

	for _, v := range request {
		for _, k := range v.keys {

			log.Printf("files=%d request_length=%f", len(v.keys), v.d.End.Sub(v.d.Start).Seconds())

			result, err := s3Client.GetObject(&s3.GetObjectInput{
				Key:    aws.String(k),
				Bucket: aws.String(S3_BUCKET),
			})
			if err != nil {
				return 0, err
			}
			defer result.Body.Close()

		loop:
			for {
				_, err = io.ReadFull(result.Body, record)
				switch {
				case err == io.EOF:
					break loop
				case err != nil:
					result.Body.Close()
					return 0, err
				}

				err = msr.Unpack(record, RECORDLEN, 0, 0)
				if err != nil {
					return 0, err
				}

				if msr.Starttime().Before(v.d.End) && msr.Endtime().After(v.d.Start) {
					n, err = w.Write(record)
					if err != nil {
						return 0, err
					}
					written += n
				}
			}
			result.Body.Close()
		}
	}

	return int64(written), nil
}

func fdsnDataselectV1Index(r *http.Request, h http.Header, b *bytes.Buffer) error {
	err := weft.CheckQuery(r, []string{"GET"}, []string{}, []string{})
	if err != nil {
		return err
	}

	h.Set("Content-Type", "text/html")
	_, err = b.Write(fdsnDataselectIndex)

	return err
}

func fdsnDataselectVersion(r *http.Request, h http.Header, b *bytes.Buffer) error {
	err := weft.CheckQuery(r, []string{"GET"}, []string{}, []string{})
	if err != nil {
		return err
	}

	h.Set("Content-Type", "text/plain")
	_, err = b.WriteString("1.1")

	return err
}

func fdsnDataselectWadl(r *http.Request, h http.Header, b *bytes.Buffer) error {
	err := weft.CheckQuery(r, []string{"GET"}, []string{}, []string{})
	if err != nil {
		return err
	}

	h.Set("Content-Type", "application/xml")
	_, err = b.Write(fdsnDataselectWadlFile)

	return err
}
