package main

import (
	"bytes"
	"fmt"
	"github.com/GeoNet/fdsn/internal/fdsn"
	"github.com/GeoNet/kit/mseed"
	"github.com/GeoNet/weft"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/golang/groupcache"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
	"strings"
	"time"
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

// fdsnDataselectV1Handler handles all dataselect queries.
// NRT (Near Real Time) data from the last 48 hours is fetched from the DB via a RAM cache.
// Data older than 48 hours is fetched from S3.
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

	// determine if:
	// 1. the data query is totally within the last 48 hours
	// 2. older than the last 48 hours
	// 3. crosses the 48 hour boundary
	// and serve it appropriately.

	var s []fdsn.DataSearch
	var nrtOnly = true
	var archiveOnly = true

	nrtBoundary := time.Now().UTC().Add(time.Hour * -48)

	for _, v := range params {
		s = append(s, v.Regexp())
		if v.StartTime.Before(nrtBoundary) {
			nrtOnly = false
		}
		if v.EndTime.After(nrtBoundary) {
			archiveOnly = false
		}
	}

	switch {
	case nrtOnly:
		return nrtDataOnly(s, w)
	case archiveOnly:
		return archiveDataOnly(s, w)
	default:
		return archiveNRTData(s, w)
	}
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

// nrtDataOnly serves near real time miniSEED data with RAM caching.
func nrtDataOnly(s []fdsn.DataSearch, w http.ResponseWriter) *weft.Result {
	w.Header().Set("Content-Type", "application/vnd.fdsn.mseed")

	var err error
	var keys []string
	var rec []byte

	for _, v := range s {
		keys, err = holdingsSearchNrt(v)
		if err != nil {
			return weft.InternalServerError(err)
		}
		for _, k := range keys {
			err = recordCache.Get(nil, k, groupcache.AllocatingByteSliceSink(&rec))
			switch err {
			case nil:
				w.Write(rec)
			case errNoData:
			// do nothing for no data, it could be deleted from the db
			// before we get a chance to request it.
			default:
				return weft.InternalServerError(err)
			}
		}
	}

	return &weft.StatusOK
}

// archiveDataOnly serves miniSEED data from the S3 archive.
func archiveDataOnly(s []fdsn.DataSearch, w http.ResponseWriter) *weft.Result {
	request, res := listDataFiles(s)
	if !res.Ok {
		return res
	}

	w.Header().Set("Content-Type", "application/vnd.fdsn.mseed")

	// Fetch the miniSEED files from S3.  Parse them and write
	// the records inside the time window for the query to the client.
	msr := mseed.NewMSRecord()
	defer mseed.FreeMSRecord(msr)

	record := make([]byte, RECORDLEN)

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

// archiveNRTData serves both near real time and archive data.
func archiveNRTData(s []fdsn.DataSearch, w http.ResponseWriter) *weft.Result {
	request, res := listDataFiles(s)
	if !res.Ok {
		return res
	}

	_ = request

	w.Header().Set("Content-Type", "application/vnd.fdsn.mseed")

	// seen is used to track individual miniSEED records for deduping when writing to the client.
	//  The key is network_station_location_channel_start
	var seen = make(map[string]bool)

	// Write the NRT data.

	var err error
	var keys []string
	var rec []byte

	for _, v := range s {
		keys, err = holdingsSearchNrt(v)
		if err != nil {
			return weft.InternalServerError(err)
		}
		for _, k := range keys {
			err = recordCache.Get(nil, k, groupcache.AllocatingByteSliceSink(&rec))
			switch err {
			case nil:
				w.Write(rec)
				seen[k] = true
			case errNoData:
			// do nothing for no data, it could be deleted from the db
			// before we get a chance to request it.
			default:
				return weft.InternalServerError(err)
			}
		}
	}

	// Write the archive data from S3 deduping if a record was already written as NRT data.

	// Fetch the miniSEED files from S3.  Parse them and write
	// the records inside the time window for the query to the client.
	msr := mseed.NewMSRecord()
	defer mseed.FreeMSRecord(msr)

	record := make([]byte, RECORDLEN)

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

				recordKey := fmt.Sprintf("%s_%s_%s_%s_%s", strings.Trim(msr.Network(), "\x00"),
					strings.Trim(msr.Station(), "\x00"),
					strings.Trim(msr.Channel(), "\x00"),
					strings.Trim(msr.Location(), "\x00"),
					msr.Starttime().Format(time.RFC3339Nano))

				if !seen[recordKey] && msr.Starttime().After(v.d.Start) && msr.Endtime().Before(v.d.End) {
					w.Write(record)
				}
			}
			result.Body.Close()
		}
	}

	return &weft.StatusOK
}

// listDataFiles searches the holdings DB for miniSEED files to fetch from S3.
// return an error if this would be to many files.
func listDataFiles(s []fdsn.DataSearch) ([]dataSelect, *weft.Result) {
	var request []dataSelect
	var files int

	for _, v := range s {
		keys, err := holdingsSearch(v)
		if err != nil {
			return []dataSelect{}, weft.InternalServerError(err)
		}

		files += len(keys)

		if files > MAX_FILES {
			return []dataSelect{}, &weft.Result{Code: http.StatusRequestEntityTooLarge,
				Msg: fmt.Sprintf("Number of files found: %d exceeded the limit: %d", files, MAX_FILES)}
		}

		request = append(request, dataSelect{d: v, keys: keys})
	}

	if files == 0 {
		return []dataSelect{}, &weft.Result{Ok: false, Code: http.StatusNoContent, Msg: "No results for specified query"}
	}

	return request, &weft.StatusOK
}
