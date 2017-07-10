package main

import (
	"bytes"
	"fmt"
	"github.com/GeoNet/fdsn/internal/fdsn"
	"github.com/GeoNet/fdsn/internal/weft"
	"github.com/GeoNet/kit/mseed"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
	"strings"
)

const (
	// miniSEED record length
	RECORDLEN int = 512
	// the maximum number of queries in a POST request
	MAX_QUERIES int = 1000
	// Limit the number of input files (each file is max ~10 MB).
	MAX_FILES int = 20000
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
	fdsnDataselectWadlFile, err = ioutil.ReadFile("assets/fdsn-ws-dataselect.wadl")
	if err != nil {
		log.Printf("error reading assets/fdsn-ws-dataselect.wadl: %s", err.Error())
	}

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

// fdsnDataselectV1Handler handles all dataselect queries.  It searches for matching keys in S3 and
// fetches them, writing matching records to w in the same order they were requested.
// Results are streamed to the client so a 200 can still be followed by errors which will not
// be reported to the client.  The potentially large response sizes make this the simplest solution.
func fdsnDataselectV1Handler(r *http.Request, w http.ResponseWriter) *weft.Result {
	var params []fdsn.DataSelect

	switch r.Method {
	case "POST":
		defer r.Body.Close()
		if err := fdsn.ParseDataSelectPost(r.Body, &params); err != nil {
			return weft.BadRequest(err.Error())
		}
	case "GET":
		d, err := fdsn.ParseDataSelectGet(r.URL.Query())
		if err != nil {
			return weft.BadRequest(err.Error())
		}

		params = append(params, d)
	default:
		return &weft.MethodNotAllowed
	}

	if len(params) > MAX_QUERIES {
		return &weft.Result{Code: http.StatusRequestEntityTooLarge,
			Msg: fmt.Sprintf("Number of queries in the POST request: %d exceeded the limit: %d", len(params), MAX_QUERIES)}
	}

	// search the holdings DB for the files to fetch from S3.
	// return an error if this would be to many files.
	var request []dataSelect
	var files int

	for _, v := range params {
		d := v.Regexp()

		keys, err := holdingsSearch(d)
		if err != nil {
			return weft.InternalServerError(err)
		}

		files += len(keys)

		if files > MAX_FILES {
			return &weft.Result{Code: http.StatusRequestEntityTooLarge,
				Msg: fmt.Sprintf("Number of files found: %d exceeded the limit: %d", files, MAX_FILES)}
		}

		request = append(request, dataSelect{d: d, keys: keys})
	}

	if files == 0 {
		return &weft.Result{Ok: false, Code: params[0].NoData, Msg: "No results for specified query"}
	}

	// Fetch the miniSEED files from S3.  Parse them and write
	// the records inside the time window for the query to the client.
	msr := mseed.NewMSRecord()
	defer mseed.FreeMSRecord(msr)

	record := make([]byte, RECORDLEN)

	w.Header().Set("Content-Type", "application/vnd.fdsn.mseed")

	for _, v := range request {
		for _, k := range v.keys {
			result, err := s3Client.GetObject(&s3.GetObjectInput{
				Key:    aws.String(k),
				Bucket: aws.String(S3_BUCKET),
			})
			if err != nil {
				return weft.InternalServerError(err)
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
					return weft.InternalServerError(err)
				}

				err = msr.Unpack(record, RECORDLEN, 0, 0)
				if err != nil {
					return weft.InternalServerError(err)
				}

				if msr.Starttime().After(v.d.Start) && msr.Endtime().Before(v.d.End) {
					w.Write(record)
				}
			}
			result.Body.Close()
		}
	}

	return &weft.StatusOK
}

func fdsnDataselectV1Index(r *http.Request, h http.Header, b *bytes.Buffer) *weft.Result {
	switch r.Method {
	case "GET":
		if res := weft.CheckQuery(r, []string{}, []string{}); !res.Ok {
			return res
		}

		h.Set("Content-Type", "text/html")
		b.Write(fdsnDataselectIndex)
		return &weft.StatusOK
	default:
		return &weft.MethodNotAllowed
	}
}

func fdsnDataselectVersion(r *http.Request, h http.Header, b *bytes.Buffer) *weft.Result {
	switch r.Method {
	case "GET":
		if res := weft.CheckQuery(r, []string{}, []string{}); !res.Ok {
			return res
		}

		h.Set("Content-Type", "text/plain")
		b.WriteString("1.1")
		return &weft.StatusOK
	default:
		return &weft.MethodNotAllowed
	}
}

func fdsnDataselectWadl(r *http.Request, h http.Header, b *bytes.Buffer) *weft.Result {
	switch r.Method {
	case "GET":
		if res := weft.CheckQuery(r, []string{}, []string{}); !res.Ok {
			return res
		}

		h.Set("Content-Type", "application/xml")
		b.Write(fdsnDataselectWadlFile)
		return &weft.StatusOK
	default:
		return &weft.MethodNotAllowed
	}
}
