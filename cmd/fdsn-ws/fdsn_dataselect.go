package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"text/template"
	"time"

	"github.com/GeoNet/fdsn/internal/fdsn"
	"github.com/GeoNet/kit/aws/s3"
	"github.com/GeoNet/kit/metrics"
	ms "github.com/GeoNet/kit/seis/ms"
	"github.com/GeoNet/kit/weft"
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
	s3Client               *s3.S3
	fdsnDataselectWadlFile []byte
	fdsnDataselectIndex    []byte
)

type dataSelect struct {
	d    fdsn.DataSearch
	keys []string
}

func initDataselectTemplate() {
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

	fdsnDataselectIndex, err = os.ReadFile("assets/fdsn-ws-dataselect.html")
	if err != nil {
		log.Printf("error reading assets/fdsn-ws-dataselect.html: %s", err.Error())
	}

	s3c, err := s3.NewWithMaxRetries(3)
	if err != nil {
		log.Fatalf("error creating S3 client: %s", err)
	}
	s3Client = &s3c
}

// fdsnDataMetricsV1Handler handles all datametrics queries.
func fdsnDataMetricsV1Handler(r *http.Request, h http.Header, b *bytes.Buffer) error {
	var params []fdsn.DataSelect

	tm := time.Now()

	switch r.Method {
	case "POST":
		defer r.Body.Close()
		if err := fdsn.ParseDataSelectPost(r.Body, &params); err != nil {
			return fdsnError{StatusError: weft.StatusError{Code: http.StatusBadRequest, Err: err}, url: r.URL.String(), timestamp: tm}
		}
	case "GET":
		d, err := fdsn.ParseDataSelectGet(r.URL.Query())
		if err != nil {
			return fdsnError{StatusError: weft.StatusError{Code: http.StatusBadRequest, Err: err}, url: r.URL.String(), timestamp: tm}
		}

		params = append(params, d)
	default:
		return fdsnError{StatusError: weft.StatusError{Code: http.StatusMethodNotAllowed}, url: r.URL.String(), timestamp: tm}
	}

	if len(params) > MAX_QUERIES {
		return fdsnError{StatusError: weft.StatusError{Code: http.StatusRequestEntityTooLarge, Err: fmt.Errorf("number of queries in the POST request: %d exceeded the limit: %d", len(params), MAX_QUERIES)}, url: r.URL.String(), timestamp: tm}
	}

	var metrics []metric

	for _, v := range params {
		d, err := v.Regexp()
		if err != nil {
			// regular expression check failed: invalid request
			return fdsnError{StatusError: weft.StatusError{Code: http.StatusBadRequest, Err: err}, url: r.URL.String(), timestamp: tm}
		}
		m, err := metricsSearch(d)
		if err != nil {
			// search into db failed: 500
			return fdsnError{StatusError: weft.StatusError{Code: http.StatusInternalServerError, Err: err}, url: r.URL.String(), timestamp: tm}
		}

		if len(metrics) > MAX_FILES {
			return fdsnError{StatusError: weft.StatusError{Code: http.StatusRequestEntityTooLarge,
				Err: fmt.Errorf("number of queries in the POST request: %d exceeded the limit: %d", len(params), MAX_QUERIES)}, url: r.URL.String(), timestamp: tm}
		}

		metrics = append(metrics, m...)
	}

	if len(metrics) == 0 {
		return fdsnError{StatusError: weft.StatusError{Code: params[0].NoData, Err: fmt.Errorf("%s", "no results for specified query")}, url: r.URL.String(), timestamp: tm}
	}

	by, err := json.Marshal(metrics)
	if err != nil {
		return fdsnError{StatusError: weft.StatusError{Code: http.StatusInternalServerError, Err: err}, url: r.URL.String(), timestamp: tm}
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

	tm := time.Now()

	switch r.Method {
	case "POST":
		defer r.Body.Close()
		if err := fdsn.ParseDataSelectPost(r.Body, &params); err != nil {
			return 0, fdsnError{StatusError: weft.StatusError{Code: http.StatusBadRequest, Err: err}, url: r.URL.String(), timestamp: tm}
		}
		if len(params) == 0 {
			return 0, fdsnError{StatusError: weft.StatusError{Code: http.StatusBadRequest, Err: fmt.Errorf("%s", "unable to parse post request")}, url: r.URL.String(), timestamp: tm}
		}
	case "GET":
		d, err := fdsn.ParseDataSelectGet(r.URL.Query())
		if err != nil {
			return 0, fdsnError{StatusError: weft.StatusError{Code: http.StatusBadRequest, Err: err}, url: r.URL.String(), timestamp: tm}
		}

		params = append(params, d)
	default:
		return 0, fdsnError{StatusError: weft.StatusError{Code: http.StatusMethodNotAllowed}, url: r.URL.String(), timestamp: tm}
	}

	if len(params) > MAX_QUERIES {
		return 0, fdsnError{StatusError: weft.StatusError{Code: http.StatusRequestEntityTooLarge,
			Err: fmt.Errorf("number of queries in the POST request: %d exceeded the limit: %d", len(params), MAX_QUERIES)}, url: r.URL.String(), timestamp: tm}
	}

	//Log extra information about POST request if needed
	if r.Method == "POST" && LOG_EXTRA {
		log.Printf("About to execute the following query params: %+v\n", params)
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
			return 0, fdsnError{StatusError: weft.StatusError{Code: http.StatusBadRequest, Err: err}, url: r.URL.String(), timestamp: tm}
		}
		if !d.End.After(d.Start) {
			return 0, fdsnError{StatusError: weft.StatusError{Code: http.StatusBadRequest, Err: fmt.Errorf("endtime must be after starttime")}, url: r.URL.String(), timestamp: tm}
		}
		// we only do "NZ"
		if m, err := regexp.MatchString(d.Network, "NZ"); err != nil || !m {
			continue
		}
		// only run query when the pattern contains only uppercase alphabetic, numbers, wildcard chars
		// if the pattern string is out of this range, we knew it won't produce results
		if fdsn.WillBeEmpty(d.Station) || fdsn.WillBeEmpty(d.Location) || fdsn.WillBeEmpty(d.Channel) {
			continue
		}
		keys, err := holdingsSearch(d)
		if err != nil {
			return 0, fdsnError{StatusError: weft.StatusError{Code: http.StatusInternalServerError, Err: err}, url: r.URL.String(), timestamp: tm}
		}

		files += len(keys)

		if files > MAX_FILES && gtHalfHour {
			return 0, fdsnError{StatusError: weft.StatusError{Code: http.StatusRequestEntityTooLarge,
				Err: fmt.Errorf("number of queries in the POST request: %d exceeded the limit: %d", len(params), MAX_QUERIES)}, url: r.URL.String(), timestamp: tm}
		}

		request = append(request, dataSelect{d: d, keys: keys})
	}

	if files == 0 {
		return 0, fdsnError{StatusError: weft.StatusError{Code: params[0].NoData, Err: fmt.Errorf("%s", "no results for specified query")}, url: r.URL.String(), timestamp: tm}
	}

	// Fetch the miniSEED files from S3.  Parse them and write
	// the records inside the time window for the query to the client.
	record := make([]byte, RECORDLEN)

	w.Header().Set("Content-Type", "application/vnd.fdsn.mseed")

	var n int
	var written int

	for _, v := range request {
		for _, k := range v.keys {

			log.Printf("files=%d request_length=%f", len(v.keys), v.d.End.Sub(v.d.Start).Seconds())

			buf := &bytes.Buffer{}
			err := s3Client.Get(S3_BUCKET, k, "", buf)
			if err != nil {
				return 0, fdsnError{StatusError: weft.StatusError{Code: http.StatusInternalServerError, Err: err}, url: r.URL.String(), timestamp: tm}
			}

		loop:
			for {
				_, err = io.ReadFull(buf, record)
				switch {
				case err == io.EOF:
					break loop
				case err != nil:
					return 0, fdsnError{StatusError: weft.StatusError{Code: http.StatusInternalServerError, Err: err}, url: r.URL.String(), timestamp: tm}
				}

				msr, err := ms.NewRecord(record)
				if err != nil {
					return 0, fdsnError{StatusError: weft.StatusError{Code: http.StatusInternalServerError, Err: err}, url: r.URL.String(), timestamp: tm}
				}

				if msr.StartTime().Before(v.d.End) && msr.EndTime().After(v.d.Start) {
					n, err = w.Write(record)
					if err != nil {
						return 0, fdsnError{StatusError: weft.StatusError{Code: http.StatusInternalServerError, Err: err}, url: r.URL.String(), timestamp: tm}
					}
					metrics.MsgTx()
					written += n
				}
			}
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
