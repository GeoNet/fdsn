package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"sort"
	"time"

	"github.com/GeoNet/fdsn/internal/fdsn"
	"github.com/GeoNet/kit/aws/s3"
	ms "github.com/GeoNet/kit/seis/ms"
	"github.com/GeoNet/kit/weft"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
)

const (
	// miniSEED record length
	RECORDLEN int = 512
)

const s3Region = "ap-southeast-2"
const s3Bucket = "geonet-open-data"
const s3Prefix = "waveforms/miniseed"

var (
	s3Client    *s3.S3        // kit client — used for ListAll
	s3RawClient *awss3.Client // raw client — used for streaming GetObject
)

// initS3Client creates S3 clients using anonymous credentials.
// geonet-open-data is a public bucket; no AWS credentials are required.
func initS3Client() error {
	anonOpt := func(o *awss3.Options) {
		o.Credentials = aws.AnonymousCredentials{}
	}

	// Build raw AWS config for the streaming client.
	cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(s3Region),
		awsconfig.WithCredentialsProvider(aws.AnonymousCredentials{}),
	)
	if err != nil {
		return fmt.Errorf("loading AWS config: %w", err)
	}
	s3RawClient = awss3.NewFromConfig(cfg)

	// Kit's getConfig() requires AWS_REGION to be set as an env var.
	os.Setenv("AWS_REGION", s3Region)

	// Build kit client for listing.
	c, err := s3.NewWithOptions(anonOpt)
	if err != nil {
		return fmt.Errorf("creating S3 client: %w", err)
	}
	s3Client = &c

	return nil
}

// fdsnDataselectV1Handler handles FDSN dataselect queries.
// It derives S3 object keys from the NSLC parameters and time range,
// fetches the miniSEED files in sorted order, and streams matching records
// directly to the client as they are found.
func fdsnDataselectV1Handler(r *http.Request, w http.ResponseWriter) (int64, error) {
	var params []fdsn.DataSelect

	tm := time.Now()

	switch r.Method {
	case "POST":
		defer func() { _ = r.Body.Close() }()
		if err := fdsn.ParseDataSelectPost(r.Body, &params); err != nil {
			return 0, fdsnError{StatusError: weft.StatusError{Code: http.StatusBadRequest, Err: err}, url: r.URL.String(), timestamp: tm}
		}
		if len(params) == 0 {
			return 0, fdsnError{StatusError: weft.StatusError{Code: http.StatusBadRequest, Err: fmt.Errorf("unable to parse post request")}, url: r.URL.String(), timestamp: tm}
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

	if r.Method == "POST" && LOG_EXTRA {
		log.Printf("About to execute the following query params: %+v\n", params)
	}

	type dataSelect struct {
		d    fdsn.DataSearch
		keys []string
	}

	var request []dataSelect
	var totalFiles int

	for _, v := range params {
		d, err := v.Regexp()
		if err != nil {
			return 0, fdsnError{StatusError: weft.StatusError{Code: http.StatusBadRequest, Err: err}, url: r.URL.String(), timestamp: tm}
		}
		if !d.End.After(d.Start) {
			return 0, fdsnError{StatusError: weft.StatusError{Code: http.StatusBadRequest, Err: fmt.Errorf("endtime must be after starttime")}, url: r.URL.String(), timestamp: tm}
		}

		keys, err := s3KeysForSearch(d)
		if err != nil {
			return 0, fdsnError{StatusError: weft.StatusError{Code: http.StatusInternalServerError, Err: err}, url: r.URL.String(), timestamp: tm}
		}

		totalFiles += len(keys)
		request = append(request, dataSelect{d: d, keys: keys})
	}

	if totalFiles == 0 {
		return 0, fdsnError{StatusError: weft.StatusError{Code: params[0].NoData, Err: fmt.Errorf("no results for specified query")}, url: r.URL.String(), timestamp: tm}
	}

	record := make([]byte, RECORDLEN)
	w.Header().Set("Content-Type", "application/vnd.fdsn.mseed")

	flusher, canFlush := w.(http.Flusher)
	var written int

	for _, v := range request {
		for _, k := range v.keys {
			// Check file time span via ranged header fetches before downloading the full file.
			overlaps, err := fileOverlapsWindow(k, v.d.Start, v.d.End)
			if err != nil {
				log.Printf("skipping key=%s: header check failed: %v", k, err)
				continue
			}
			if !overlaps {
				log.Printf("skipping key=%s: no overlap with query window", k)
				continue
			}

			log.Printf("fetching key=%s start=%s end=%s", k, v.d.Start.Format(time.RFC3339), v.d.End.Format(time.RFC3339))

			result, err := s3RawClient.GetObject(context.Background(), &awss3.GetObjectInput{
				Bucket: aws.String(s3Bucket),
				Key:    aws.String(k),
			})
			if err != nil {
				// Key not found — this day/NSLC file simply doesn't exist.
				log.Printf("miniSEED file not found, key: %s (%v)", k, err)
				continue
			}

		loop:
			for {
				_, err = io.ReadFull(result.Body, record)
				switch {
				case err == io.EOF:
					break loop
				case err != nil:
					result.Body.Close()
					return 0, fdsnError{StatusError: weft.StatusError{Code: http.StatusInternalServerError, Err: err}, url: r.URL.String(), timestamp: tm}
				}

				msr, err := ms.NewRecord(record)
				if err != nil {
					result.Body.Close()
					return 0, fdsnError{StatusError: weft.StatusError{Code: http.StatusInternalServerError, Err: err}, url: r.URL.String(), timestamp: tm}
				}

				if msr.StartTime().Before(v.d.End) && msr.EndTime().After(v.d.Start) {
					n, err := w.Write(record)
					if err != nil {
						result.Body.Close()
						return 0, fdsnError{StatusError: weft.StatusError{Code: http.StatusInternalServerError, Err: err}, url: r.URL.String(), timestamp: tm}
					}
					written += n
					if canFlush {
						flusher.Flush()
					}
				}
			}
			result.Body.Close()
		}
	}

	if written == 0 {
		return 0, fdsnError{StatusError: weft.StatusError{Code: http.StatusNoContent, Err: fmt.Errorf("no results for specified query")}, url: r.URL.String(), timestamp: tm}
	}

	return int64(written), nil
}

// fileOverlapsWindow fetches only the first and last 512-byte miniSEED records
// from the S3 file using ranged GET requests, and reports whether the file's
// time span overlaps [start, end). This avoids downloading the full file when
// it falls entirely outside the query window.
func fileOverlapsWindow(key string, start, end time.Time) (bool, error) {
	// check file size, if the size is less than 2K then don't bother checking this
	head, err := s3RawClient.HeadObject(context.Background(), &awss3.HeadObjectInput{
		Bucket: aws.String(s3Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return false, fmt.Errorf("head object: %w", err)
	}
	if head.ContentLength != nil && *head.ContentLength < int64(2*RECORDLEN) {
		return true, nil
	}

	// First record → file start time.
	buf := &bytes.Buffer{}
	if err := s3Client.GetByteRange(s3Bucket, key, "", "bytes=0-511", buf); err != nil {
		return false, fmt.Errorf("range fetch first record: %w", err)
	}
	firstRec, err := ms.NewRecord(buf.Bytes())
	if err != nil {
		return false, fmt.Errorf("parsing first record: %w", err)
	}
	fileStart := firstRec.StartTime()

	// Last record → file end time.
	buf.Reset()
	if err := s3Client.GetByteRange(s3Bucket, key, "", "bytes=-512", buf); err != nil {
		return false, fmt.Errorf("range fetch last record: %w", err)
	}
	lastRec, err := ms.NewRecord(buf.Bytes())
	if err != nil {
		return false, fmt.Errorf("parsing last record: %w", err)
	}
	fileEnd := lastRec.EndTime()

	return fileStart.Before(end) && fileEnd.After(start), nil
}

// s3KeysForSearch returns all S3 keys matching the DataSearch parameters,
// sorted by key path so files are fetched in chronological order.
//
// The S3 path structure is:
//
//	waveforms/miniseed/{yyyy}/{yyyy}.{doy}/{station}.{network}/{yyyy}.{doy}.{station}.{location}-{channel}.{network}.D
func s3KeysForSearch(d fdsn.DataSearch) ([]string, error) {
	netRe, err := regexp.Compile(d.Network)
	if err != nil {
		return nil, fmt.Errorf("invalid network pattern: %w", err)
	}
	staRe, err := regexp.Compile(d.Station)
	if err != nil {
		return nil, fmt.Errorf("invalid station pattern: %w", err)
	}
	locRe, err := regexp.Compile(d.Location)
	if err != nil {
		return nil, fmt.Errorf("invalid location pattern: %w", err)
	}
	chRe, err := regexp.Compile(d.Channel)
	if err != nil {
		return nil, fmt.Errorf("invalid channel pattern: %w", err)
	}

	seen := make(map[string]struct{})
	var keys []string

	// Start one day before the query window to capture records whose miniSEED
	// file is dated the previous day but whose data overlaps the query start
	// (matching the -24h lookback used by fdsn-ws when querying the holdings DB).
	startDay := d.Start.UTC().Truncate(24*time.Hour).AddDate(0, 0, -1)
	// End at the day that contains d.End. If d.End falls exactly on midnight,
	// that day's files start at or after d.End and cannot overlap the window,
	// so we stop at the previous day.
	endDay := d.End.UTC().Truncate(24 * time.Hour)
	if d.End.UTC().Equal(endDay) {
		endDay = endDay.AddDate(0, 0, -1)
	}

	for t := startDay; !t.After(endDay); t = t.AddDate(0, 0, 1) {
		yyyy := t.Format("2006")
		doy := fmt.Sprintf("%03d", t.YearDay())
		dayPrefix := fmt.Sprintf("%s/%s/%s.%s/", s3Prefix, yyyy, yyyy, doy)

		candidates, err := s3Client.ListAll(s3Bucket, dayPrefix)
		if err != nil {
			return nil, fmt.Errorf("S3 list error for prefix %s: %w", dayPrefix, err)
		}

		for _, key := range candidates {
			net, sta, loc, ch, ok := parseS3Key(key)
			if !ok {
				continue
			}
			if !netRe.MatchString(net) || !staRe.MatchString(sta) || !locRe.MatchString(loc) || !chRe.MatchString(ch) {
				continue
			}
			if _, dup := seen[key]; !dup {
				seen[key] = struct{}{}
				keys = append(keys, key)
			}
		}
	}

	sort.Strings(keys)
	return keys, nil
}

// s3KeyRegexp matches the open-data S3 key format and captures NSLC components.
//
// Path format:
//
//	waveforms/miniseed/{yyyy}/{yyyy}.{doy}/{station}.{network}/{yyyy}.{doy}.{station}.{location}-{channel}.{network}.D
//
// Capture groups: [1]=yyyy [2]=yyyy(dir) [3]=doy(dir) [4]=sta(dir) [5]=net(dir)
//
//	[6]=yyyy(file) [7]=doy(file) [8]=station [9]=location [10]=channel [11]=network
var s3KeyRegexp = regexp.MustCompile(
	`^waveforms/miniseed/` +
		`(\d{4})/` +
		`(\d{4})\.(\d{3})/` +
		`([^/]+)\.([A-Z0-9]+)/` +
		`(\d{4})\.(\d{3})\.([^.]+)\.([^-]*)-([^.]+)\.([A-Z0-9]+)\.D$`)

// parseS3Key extracts network, station, location, and channel from an S3 key.
// Returns ok=false if the key does not match the expected pattern.
func parseS3Key(key string) (network, station, location, channel string, ok bool) {
	m := s3KeyRegexp.FindStringSubmatch(key)
	if m == nil {
		return "", "", "", "", false
	}
	// m[8]=station, m[9]=location, m[10]=channel, m[11]=network (all from filename segment)
	return m[11], m[8], m[9], m[10], true
}

func fdsnDataselectV1Index(r *http.Request, h http.Header, b *bytes.Buffer) error {
	if err := weft.CheckQuery(r, []string{"GET"}, []string{}, []string{}); err != nil {
		return err
	}
	h.Set("Content-Type", "text/html")
	b.Write(dataselectIndexHTML)
	return nil
}

func fdsnDataselectVersion(r *http.Request, h http.Header, b *bytes.Buffer) error {
	if err := weft.CheckQuery(r, []string{"GET"}, []string{}, []string{}); err != nil {
		return err
	}
	h.Set("Content-Type", "text/plain")
	b.WriteString(dataselectVersion)
	return nil
}

func fdsnDataselectWadl(r *http.Request, h http.Header, b *bytes.Buffer) error {
	if err := weft.CheckQuery(r, []string{"GET"}, []string{}, []string{}); err != nil {
		return err
	}
	h.Set("Content-Type", "application/xml")
	b.Write(dataselectWADL)
	return nil
}

var dataselectIndexHTML = []byte(`<!doctype html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <title>FDSNWS - Dataselect (Open Data Proxy)</title>
</head>
<body>
<h1>FDSNWS Dataselect Web Service (Open Data Proxy)</h1>
<p>The dataselect Web service provides access to waveform data in
<a href="https://ds.iris.edu/ds/nodes/dmc/data/formats/miniseed/">miniseed</a> format
sourced from the GeoNet open data S3 bucket.</p>

<h2>Available URLs</h2>
<ul>
    <li><a href="/fdsnws/dataselect/1/query">query</a></li>
    <li><a href="/fdsnws/dataselect/1/version">version</a></li>
    <li><a href="/fdsnws/dataselect/1/application.wadl">application.wadl</a></li>
</ul>

<h2>Feature Notes</h2>
<ul>
    <li>Data is sourced from s3://geonet-open-data/waveforms/miniseed/</li>
</ul>
</body>
</html>`)

var dataselectWADL = []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<application xmlns="http://wadl.dev.java.net/2009/02"
        xmlns:xsd="http://www.w3.org/2001/XMLSchema">
    <resources base="/fdsnws/dataselect/1/">
        <resource path="query">
            <method href="#queryGET"/>
            <method href="#queryPOST"/>
        </resource>
        <resource path="version">
            <method name="GET">
                <response>
                    <representation mediaType="text/plain"/>
                </response>
            </method>
        </resource>
        <resource path="application.wadl">
            <method name="GET">
                <response>
                    <representation mediaType="application/xml"/>
                </response>
            </method>
        </resource>
    </resources>
    <method name="GET" id="queryGET">
        <request>
            <param name="starttime" style="query" type="xsd:dateTime"/>
            <param name="endtime" style="query" type="xsd:dateTime"/>
            <param name="network" style="query" type="xsd:string"/>
            <param name="station" style="query" type="xsd:string"/>
            <param name="location" style="query" type="xsd:string"/>
            <param name="channel" style="query" type="xsd:string"/>
            <param name="format" style="query" type="xsd:string" default="miniseed">
                <option value="miniseed"/>
            </param>
            <param name="nodata" style="query" type="xs:int" default="204">
                <option value="204"/>
                <option value="404"/>
            </param>
        </request>
        <response status="200">
            <representation mediaType="application/vnd.fdsn.mseed"/>
        </response>
        <response status="204 400 401 403 404 413 414 500 503">
            <representation mediaType="text/plain; charset=utf-8"/>
        </response>
    </method>
    <method name="POST" id="queryPOST">
        <response status="200">
            <representation mediaType="application/vnd.fdsn.mseed"/>
        </response>
        <response status="204 400 401 403 404 413 414 500 503">
            <representation mediaType="text/plain; charset=utf-8"/>
        </response>
    </method>
</application>`)
