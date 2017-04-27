package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/GeoNet/collect/mseed"
	"github.com/GeoNet/weft"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"time"
)

const (
	// the record length of the miniseed records.  Constant for all GNS miniseed files
	RECORDLEN int = 512
	// the number of times to retry downloading a file from S3
	MAX_RETRIES int = 5
	// Currently the largest file in the bucket is ~10 MB so downloads in a few seconds, especially from S3 to EC2, so use this timeout
	// to stop and retry the download
	FETCH_TIMEOUT time.Duration = time.Minute
	HTTP_TIMEOUT  time.Duration = FETCH_TIMEOUT * time.Duration(MAX_RETRIES)
	// the max number of worker goroutines downloading files from S3 in parallel.
	// See https://github.com/aws/aws-sdk-go/issues/190 for detail on why we keep this number small
	MAX_WORKERS int = 10
	// the maximum number of queries in a POST request
	MAX_QUERIES int = 1000
	// Limit the number of input files (each file is max ~10 MB).  We can handle a large number so could increase or remove this limit.
	MAX_FILES int64 = 20000
)

var (
	fdsnDataselectWadlFile []byte
	fdsnDataselectIndex    []byte
)

type fdsnDataselectV1 struct {
	StartTime Time     `schema:"starttime"` // limit to data on or after the specified start time.
	EndTime   Time     `schema:"endtime"`   // limit to data on or before the specified end time.
	Network   []string `schema:"network"`   // network name of data to query
	Station   []string `schema:"station"`   // station name of data to query
	Location  []string `schema:"location"`  // location name of data to query
	Channel   []string `schema:"channel"`   // channel number of data to query
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
}

// fdsnDataselectV1Handler handles all dataselect queries.  It searches for matching keys in S3 and
// fetches them in parallel, writing matching records to w in the same order they were requested.
// This parses all input files before writing the StatusCode and before writing data to ResponseWriter.
// In the case of an error a non-200 status code is returned as a weft.Result and no output written to w.
func fdsnDataselectV1Handler(r *http.Request, h http.Header, w http.ResponseWriter) *weft.Result {
	params, res := dataSelectParams(r)
	if res != nil {
		return res
	}

	// Get a list of files to read from S3 (and possibly other sources) that match the query.  Iterate over the records in the files,
	// writing the records that match.
	var err error
	var matches []match
	ctx := r.Context()
	if matches, err = matchingKeys(ctx, params); err != nil {
		return weft.BadRequest(err.Error())
	}

	nFiles := len(matches)
	if nFiles == 0 {
		return &weft.Result{Ok: false, Code: http.StatusNoContent, Msg: "No results for specified query"}
	}

	if int64(nFiles) >= MAX_FILES {
		message := fmt.Sprintf("Too many files in request:%d, max: %d", nFiles, MAX_FILES)
		return &weft.Result{Ok: false, Code: http.StatusRequestEntityTooLarge, Msg: message}
	}

	nWorkers := MAX_WORKERS
	if nFiles < nWorkers {
		nWorkers = nFiles
	}

	for i := range matches {
		matches[i].index = i
		matches[i].buff = make(chan bytes.Buffer)
	}

	inputs := make(chan match, nFiles)
	quitWorker := make(chan bool)
	errChan := make(chan error)

	// launch N goroutines, initially blocked waiting for work.
	for i := 0; i < nWorkers; i++ {
		go worker(ctx, inputs, quitWorker, errChan)
	}

	// Send work to the input buffered chan, gets nWorkers fetching/parsing in parallel
	for _, m := range matches {
		inputs <- m
	}

	// everything worked so set headers before writing body.
	// Obspy and curl check against Content-Length header but we don't know this yet
	h.Set("Surrogate-Control", "max-age=10")
	h.Set("Content-Type", "application/vnd.fdsn.mseed")

	for _, r := range matches {
		select {
		case err := <-errChan:
			return weft.InternalServerError(err)
		case buff := <-r.buff:
			if _, err := buff.WriteTo(w); err != nil {
				quitWorker <- true
				return weft.InternalServerError(err)
			}
		}
	}

	return &weft.StatusOK
}

// worker is the function running as a goroutine that fetches and parses data until an error is seen,
// the context is cancelled, it is told to quit, or there is no more work left in the input channel.
func worker(ctx context.Context, inputs chan match, quitWorker chan bool, errChan chan error) {
	for {
		select {
		case <-quitWorker:
			return
		case input := <-inputs:
			data, err := input.fetch(ctx)
			if err != nil {
				log.Printf("error fetching file: %s, err: %s", input.key, err.Error())
				errChan <- err
				return
			}

			outBuff, err := input.parse(bytes.NewBuffer(data))
			if err != nil {
				log.Printf("error parsing file: %s, err: %s", input.key, err.Error())
				errChan <- err
				return
			}

			input.buff <- outBuff
		default:
			return
		}
	}
}

// dataSelectParams parses both POST and GET requests and returns a slice of parameters as params.
func dataSelectParams(r *http.Request) (params []fdsnDataselectV1, res *weft.Result) {
	switch r.Method {
	case "POST":
		c, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return nil, weft.BadRequest(err.Error())
		}
		defer r.Body.Close()

		var dsq dataSelectPostQuery
		if err := dsq.unmarshal(c); err != nil {
			return nil, weft.BadRequest(err.Error())
		}

		params = dsq
	case "GET":
		e := fdsnDataselectV1{}
		values := r.URL.Query()

		conv := map[string]string{
			"net":   "network",
			"sta":   "station",
			"loc":   "location",
			"cha":   "channel",
			"start": "starttime",
			"end":   "endtime",
		}

		// convert all abbreviated params to their expanded form
		for abbrev, expanded := range conv {
			if val, ok := values[abbrev]; ok {
				values[expanded] = val
				delete(values, abbrev)
			}
		}

		err := decoder.Decode(&e, values)
		if err != nil {
			return nil, weft.BadRequest(err.Error())
		}

		// Defaults: as per spec we need to include any valid files in the search so use wildcards and broad time range
		if len(e.Network) == 0 {
			e.Network = []string{"*"}
		}
		if len(e.Station) == 0 {
			e.Station = []string{"*"}
		}
		if len(e.Location) == 0 {
			e.Location = []string{"*"}
		}
		if len(e.Channel) == 0 {
			e.Channel = []string{"*"}
		}

		if e.StartTime.IsZero() {
			e.StartTime.Time, err = time.Parse(time.RFC3339, "2000-01-01T00:00:00Z")
			if err != nil {
				return nil, weft.BadRequest(err.Error())
			}
		}

		if e.EndTime.IsZero() {
			e.EndTime.Time = time.Now().UTC()
		}

		params = append(params, e)
	default:
		return nil, &weft.MethodNotAllowed
	}

	if len(params) > MAX_QUERIES {
		message := fmt.Sprintf("Number of queries in the POST request: %d exceeded the limit: %d", len(params), MAX_QUERIES)
		return nil, &weft.Result{Ok: false, Code: http.StatusRequestEntityTooLarge, Msg: message}
	}

	return params, nil
}

// matchingKeys parses the parameters and queries S3 to get a list of matching keys (files), returning an error if
// parsing fails
func matchingKeys(ctx context.Context, params []fdsnDataselectV1) (matches []match, err error) {
	var ds *s3DataSource

	for _, param := range params {
		ds, err = newS3DataSource(S3_BUCKET, param, MAX_RETRIES)
		if err != nil {
			return nil, err
		}

		keys, err := ds.matchingKeys(ctx)
		if err != nil {
			return nil, err
		}

		for _, key := range keys {
			matches = append(matches, []match{{dataSource: ds, key: key}}...)
		}
	}

	return matches, nil
}

type match struct {
	dataSource *s3DataSource
	key        string
	index      int
	buff       chan bytes.Buffer // holding the output of the buffer here, only valid when read.
}

func (m match) fetch(ctx context.Context) (b []byte, err error) {

	download := func(retry int) (bs []byte, err error) {
		// Creating a timeout context as a child of ctx.  This is required because S3 requires a single context
		// and we want to handle both cancellation and timeouts in this single context.  Running in a closure so
		// we can 'defer cancel()' instead of calling it in multiple places.
		timeoutCtx, cancel := context.WithTimeout(ctx, FETCH_TIMEOUT)
		defer cancel()

		data, err := m.dataSource.getObject(timeoutCtx, m.key)

		switch err {
		case nil:
			if retry > 0 {
				log.Printf("fetch attempt %d/%d successful for file: %s\n", retry+1, MAX_RETRIES, m.key)
			}

			return data, nil
		default:
			return bs, err
		}
	}

	for retry := 0; retry < MAX_RETRIES; retry++ {
		if b, err = download(retry); err == nil {
			return b, nil
		}

		// if the context was cancelled (eg: user cancelling the request) then return the error immediately without retrying
		select {
		case <-ctx.Done():
			return b, ctx.Err()
		default:
			log.Printf("failed to download file: %s, attempt %d/%d, err:%s\n", m.key, retry+1, MAX_RETRIES, err)
		}
	}

	return b, fmt.Errorf("failed to download file: %s, all attempts failed.  Err: %s\n", m.key, err)
}

func (m match) parse(inBuff *bytes.Buffer) (out bytes.Buffer, err error) {
	msr := mseed.NewMSRecord()
	defer mseed.FreeMSRecord(msr)

	params := m.dataSource.params
	net, sta, loc, cha, _, _ := m.dataSource.regexp()
	re := []string{net, sta, loc, cha}

loop:
	for {
		record := inBuff.Next(RECORDLEN)

		if len(record) == 0 {
			break loop
		}

		// skipping anything less than 512 since this must be the last record in the file and libmseed will probably crash.  Do we need to pad?
		if len(record) < RECORDLEN {
			return out, errors.New("last record in file was smaller than expected size")
		}

		if err := msr.Unpack(record, RECORDLEN, 0, 0); err != nil {
			return out, err
		}

		// the msr.Endtime() appears to be identical to msr.Starttime so just check the start time is within bounds
		msrStart := Time{msr.Starttime()}
		msrEnd := Time{msr.Starttime()}
		if msrStart.Before(params.StartTime.Time) || msrEnd.After(params.EndTime.Time) {
			continue
		}

		// Checking that network, station, location and channel match the input params
		var match bool
		msrFields := []string{msr.Network(), msr.Station(), msr.Location(), msr.Channel()}

		var err error
		for idx, msrField := range msrFields {
			// trim the C null terminator off
			msrField = strings.TrimRight(msrField, "\x00")

			if match, err = regexp.MatchString(re[idx], msrField); err != nil {
				return out, err
			}
			if !match {
				break
			}
		}

		if !match {
			continue
		}

		if _, err := out.Write(record); err != nil {
			return out, err
		}
	}

	return out, nil
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

type dataSelectPostQuery []fdsnDataselectV1

func (dsq *dataSelectPostQuery) unmarshal(b []byte) error {
	scanner := bufio.NewScanner(bytes.NewReader(b))
	for scanner.Scan() {

		line := scanner.Text()
		// ignore any blank lines or lines with "=", we don't use any of these parameters and "=" is otherwise invalid
		if len(line) == 0 || strings.Contains(line, "=") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) != 6 {
			return fmt.Errorf("incorrect number of fields in dataselect query POST body, expected 6 but observed: %d", len(fields))
		}

		startTime := Time{}
		if err := startTime.UnmarshalText([]byte(fields[4])); err != nil {
			return err
		}

		endTime := Time{}
		if err := endTime.UnmarshalText([]byte(fields[5])); err != nil {
			return err
		}

		*dsq = append(*dsq,
			fdsnDataselectV1{
				StartTime: startTime,
				EndTime:   endTime,
				Network:   []string{fields[0]},
				Station:   []string{fields[1]},
				Location:  []string{fields[2]},
				Channel:   []string{fields[3]},
			})
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}
