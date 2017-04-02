package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/GeoNet/collect/mseed"
	"github.com/GeoNet/weft"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
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
	MAX_FILES int64 = 3000
)

var (
	fdsnDataselectWadlFile []byte
	fdsnDataselectIndex    []byte
)

type fdsnDataselectV1 struct {
	StartTime Time   `schema:"starttime"` // limit to data on or after the specified start time.
	EndTime   Time   `schema:"endtime"`   // limit to data on or before the specified end time.
	Network   string `schema:"network"`   // network name of data to query
	Station   string `schema:"station"`   // station name of data to query
	Location  string `schema:"location"`  // location name of data to query
	Channel   string `schema:"channel"`   // channel number of data to query
}

func init() {
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
// This parses all input files before writing the StatusCode and before writing data to w.  In the case
// of an error a non-200 status code is returned as a weft.Result and no output written to w.
func fdsnDataselectV1Handler(r *http.Request, h http.Header, w http.ResponseWriter) *weft.Result {
	params, res := dataSelectParams(r)
	if res != nil {
		return res
	}

	if len(params) > MAX_QUERIES {
		message := fmt.Sprintf("Number of queries in the POST request: %d exceeded the limit: %d", len(params), MAX_QUERIES)
		return &weft.Result{Ok: false, Code: http.StatusRequestEntityTooLarge, Msg: message}
	}

	// Get a list of files to read from S3 (and possibly other sources) that match the query.  Iterate over the records in the files,
	// writing the records that match.
	var err error
	var matches []match
	for _, param := range params {
		var subset []match
		if subset, err = matchingKeys(param); err != nil {
			return weft.BadRequest(err.Error())
		}

		if len(subset) == 0 {
			message := fmt.Sprintf("No matching files for net:%s sta:%s loc:%s cha:%s starttime:%s endtime%s",
				param.Network,
				param.Station,
				param.Location,
				param.Channel,
				param.StartTime.Format(time.RFC3339),
				param.EndTime.Format(time.RFC3339),
			)
			return &weft.Result{Ok: false, Code: http.StatusNoContent, Msg: message}
		}

		matches = append(matches, subset...)
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

	ctx := r.Context()
	inputs := make(chan match, nFiles)
	quitWorker := make(chan bool)
	errChan := make(chan error)

	worker := func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-quitWorker:
				return
			case input := <-inputs:

				data, err := input.fetch(ctx)
				var buff = bytes.NewBuffer(data)
				if err != nil {
					log.Printf("error fetching file: %s, err: %s", input.key, err.Error())
					errChan <- err
					return
				}

				var outBuff bytes.Buffer
				outBuff, err = input.parse(ctx, buff)
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

	// launch N goroutines, initially blocked waiting for work.
	for i := 0; i < nWorkers; i++ {
		go worker()
	}

	// Send work to the input buffered chan, gets nWorkers fetching/parsing in parallel
	for _, m := range matches {
		inputs <- m
	}

	// Storing to a tempfile (deleted on exit) so we can see if the entire processes worked without an error, otherwise
	// it's difficult to show that an error occurred.
	tempFile, err := ioutil.TempFile("", "fdsn_tempfile")
	if err != nil {
		return weft.InternalServerError(err)
	}
	defer tempFile.Close()
	defer os.Remove(tempFile.Name())

	for _, r := range matches {
		select {
		case <-ctx.Done():
			return weft.InternalServerError(ctx.Err())
		case err := <-errChan:
			return weft.InternalServerError(err)
		case buff := <-r.buff:
			if _, err := buff.WriteTo(tempFile); err != nil {
				quitWorker <- true
				return weft.InternalServerError(err)
			}
		}
	}

	if _, err := tempFile.Seek(0, 0); err != nil {
		log.Println("error seeking to start of tempfile", err)
		return weft.InternalServerError(err)
	}

	tempFileInfo, err := tempFile.Stat()
	if err != nil {
		log.Println("error getting size of tempfile", err)
		return weft.InternalServerError(err)
	}

	// everything worked so set headers before writing body.
	// Obspy and curl check against Content-Length header as a check that the download completed
	h.Set("Surrogate-Control", "max-age=10")
	h.Set("Content-Type", "application/vnd.fdsn.mseed")
	h.Set("Content-Length", strconv.FormatInt(tempFileInfo.Size(), 10))

	if _, err := io.Copy(w, tempFile); err != nil {
		log.Printf("error copying from tempfile to ResponseWriter: err: %s", err.Error())
		return weft.InternalServerError(err)
	}

	return &weft.StatusOK
}

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

		err := decoder.Decode(&e, values)
		if err != nil {
			return nil, weft.BadRequest(err.Error())
		}

		// Defaults: as per spec we need to include any valid files in the search so use wildcards and broad time range
		if e.Network == "" {
			e.Network = "*"
		}
		if e.Station == "" {
			e.Station = "*"
		}
		if e.Location == "" {
			e.Location = "*"
		}
		if e.Channel == "" {
			e.Channel = "*"
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

	return params, nil
}

// matchingKeys parses the parameters and queries S3 to get a list of matching keys (files), returning an error if
// parsing fails
func matchingKeys(param fdsnDataselectV1) (subset []match, err error) {
	var ds *s3DataSource
	ds, err = newS3DataSource(S3_BUCKET, param, MAX_RETRIES)
	if err != nil {
		return nil, err
	}

	keys, err := ds.matchingKeys()
	if err != nil {
		return nil, err
	}

	for _, key := range keys {
		subset = append(subset, match{dataSource: ds, key: key})
	}

	return subset, nil
}

type match struct {
	dataSource *s3DataSource
	key        string
	index      int
	buff       chan bytes.Buffer // holding the output of the buffer here, only valid when read.
}

func (m match) fetch(ctx context.Context) (b []byte, err error) {

loop:
	for retry := 0; retry < MAX_RETRIES; retry++ {
		timeoutCtx, cancel := context.WithTimeout(context.Background(), FETCH_TIMEOUT)

		// downloading the file in another goroutine so we can use select to retry the download.  The existing
		// goroutine will continue until it errors from a timeout and it's output will be discarded.  Unfortunately AWS doesn't
		// support context for cancellation/timeouts.
		type chanResponse struct {
			index int
			buff  []byte
			err   error
		}

		var res = make(chan chanResponse)

		go func() {
			var err error
			data, err := m.dataSource.getObject(m.key)
			res <- chanResponse{index: m.index, buff: data, err: err}
		}()

		select {
		case r := <-res:
			cancel()
			if r.err != nil {
				if retry >= MAX_RETRIES {
					log.Println("error fetching file, max retries reached, exiting", r.err)
					return r.buff, r.err
				}

				log.Printf("error fetching file: %s, retrying (attempt no: %d/%d. err: %s\n", m.key, retry+1, MAX_RETRIES, r.err.Error())
				continue loop
			}

			// Happy path:
			if r.err == nil && retry > 0 {
				log.Printf("fetch attempt %d/%d successful for file: %s\n", retry+1, MAX_RETRIES, m.key)
			}

			return r.buff, nil
		case <-ctx.Done():
			cancel()
			return b, ctx.Err()
		case <-timeoutCtx.Done():
			cancel()
			if retry >= MAX_RETRIES {
				log.Println("timeout fetching file, max retries reached, exiting", timeoutCtx.Err().Error())
				return b, timeoutCtx.Err()
			}

			log.Printf("timeout fetching file %s, attempt number %d/%d\n", m.key, retry+1, MAX_RETRIES)
		}
	}

	return b, fmt.Errorf("error fetching file: %s\n", m.key)
}

func (m match) parse(ctx context.Context, inBuff *bytes.Buffer) (out bytes.Buffer, err error) {
	msr := mseed.NewMSRecord()
	defer mseed.FreeMSRecord(msr)

loop:
	for {
		select {
		case <-ctx.Done():
			return out, ctx.Err()
		default:
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
			params := m.dataSource.params
			if msrStart.Before(params.StartTime.Time) || msrEnd.After(params.EndTime.Time) {
				continue
			}

			// Checking that network, station, location and channel match the input params
			var match bool
			re := m.dataSource.regexp()
			msrFields := []string{msr.Network(), msr.Station(), msr.Location(), msr.Channel()}

			var err error
			for idx, msrField := range msrFields {
				// trim the null terminator off
				msrField = strings.TrimRight(msrField, "\x00")

				if match, err = regexp.MatchString(re[idx], msrField); err != nil {
					//return result{index: m.index, err: err}
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
				Network:   fields[0],
				Station:   fields[1],
				Location:  fields[2],
				Channel:   fields[3],
			})
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}
