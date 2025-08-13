package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/GeoNet/kit/weft"
)

var mux *http.ServeMux

func initRoutes() {
	mux = http.NewServeMux()

	mux.HandleFunc("/", weft.MakeHandler(weft.NoMatch, weft.TextError))
	mux.HandleFunc("/soh/up", weft.MakeHandler(weft.Up, weft.TextError))
	mux.HandleFunc("/soh", weft.MakeHandler(soh, weft.UseError))

	// fdsn-ws-event
	mux.HandleFunc("/fdsnws/event/1/", weft.MakeHandler(fdsnEventV1Index, weft.TextError))
	mux.HandleFunc("/fdsnws/event/1/query", weft.MakeHandler(fdsnEventV1Handler, fdsnErrorHandler))
	mux.HandleFunc("/fdsnws/event/1/version", weft.MakeHandler(fdsnEventVersion, weft.TextError))
	mux.HandleFunc("/fdsnws/event/1/catalogs", weft.MakeHandler(fdsnEventCatalogs, weft.TextError))
	mux.HandleFunc("/fdsnws/event/1/contributors", weft.MakeHandler(fdsnEventContributors, weft.TextError))
	mux.HandleFunc("/fdsnws/event/1/application.wadl", weft.MakeHandler(fdsnEventWadl, weft.TextError))

	// fdsn-ws-station
	mux.HandleFunc("/fdsnws/station/1/", weft.MakeHandler(fdsnStationV1Index, weft.TextError))
	mux.HandleFunc("/fdsnws/station/1/query", weft.MakeHandler(fdsnStationV1Handler, fdsnErrorHandler))
	mux.HandleFunc("/fdsnws/station/1/version", weft.MakeHandler(fdsnStationVersion, weft.TextError))
	mux.HandleFunc("/fdsnws/station/1/application.wadl", weft.MakeHandler(fdsnStationWadl, weft.TextError))

	// This service implements the dataselect spec from http://www.fdsn.org/webservices/FDSN-WS-Specifications-1.1.pdf.
	mux.HandleFunc("/fdsnws/dataselect/1/", weft.MakeHandler(fdsnDataselectV1Index, weft.TextError))
	mux.HandleFunc("/fdsnws/dataselect/1/query", weft.MakeDirectHandler(fdsnDataselectV1Handler, fdsnErrorHandler))
	mux.HandleFunc("/fdsnws/dataselect/1/version", weft.MakeHandler(fdsnDataselectVersion, weft.TextError))
	mux.HandleFunc("/fdsnws/dataselect/1/application.wadl", weft.MakeHandler(fdsnDataselectWadl, weft.TextError))

	mux.HandleFunc("/metrics/fdsnws/dataselect/1/query", weft.MakeHandler(fdsnDataMetricsV1Handler, weft.TextError))

	mux.HandleFunc("/sc3ml", weft.MakeHandler(s3ml, weft.TextError))

	// handle robots
	mux.HandleFunc("/robots.txt", weft.MakeHandler(robots, weft.TextError))
}

func soh(r *http.Request, h http.Header, b *bytes.Buffer) error {
	err := weft.CheckQuery(r, []string{"GET"}, []string{}, []string{})
	if err != nil {
		return err
	}

	var i int

	err = db.QueryRow(`SELECT 1`).Scan(&i)
	if err != nil {
		return err
	}

	b.WriteString("<html><head></head><body>ok</body></html>")

	return nil
}

const FDSN_ERR_FORMAT = `Error %03d: %s
%s
Usage details are available from https://www.geonet.org.nz/data/tools/FDSN
Request:
%s
Request Submitted:
%s
Service version:
%s`

type fdsnError struct {
	weft.StatusError
	url       string
	timestamp time.Time
}

func fdsnErrorHandler(err error, h http.Header, b *bytes.Buffer, nounce string) error {
	switch e := err.(type) {
	case fdsnError:
		var ver string
		if strings.HasPrefix(e.url, "/fdsnws/event/") {
			ver = eventVersion
		} else if strings.HasPrefix(e.url, "/fdsnws/station/") {
			ver = stationVersion
		} else if strings.HasPrefix(e.url, "fdsnws/dataselect/") {
			ver = dataselectVersion
		}

		h.Set("Content-Type", "text/plain; charset=utf-8")

		switch e.Code {
		case http.StatusNoContent, http.StatusNotFound: // NOTE: though NoContent is not an error but we handled here
			h.Set("Surrogate-Control", "max-age=10")
		case http.StatusBadRequest:
			h.Set("Surrogate-Control", "max-age=86400")
		case http.StatusMethodNotAllowed:
			h.Set("Surrogate-Control", "max-age=86400")
		case http.StatusServiceUnavailable, http.StatusInternalServerError:
			h.Set("Surrogate-Control", "max-age=10")
		default:
		}

		// "no content" can't have a http body
		if e.Code != http.StatusNoContent && e.Code != http.StatusNotFound {
			msg := fmt.Sprintf(FDSN_ERR_FORMAT, e.Code, http.StatusText(e.Code), e.Err, e.url, e.timestamp.Format(time.RFC3339), ver)
			b.WriteString(msg)
		}
		return nil
	}

	return weft.TextError(err, h, b, nounce)
}

//go:embed assets/robots.txt
var robot string

// robots handler for crawlers
func robots(r *http.Request, h http.Header, b *bytes.Buffer) error {
	h.Set("Content-Type", "text/plain")
	h.Set("Surrogate-Control", "max-age=3600")

	_, err := b.WriteString(robot)
	if err != nil {
		return weft.StatusError{Code: http.StatusInternalServerError, Err: err}
	}
	return nil
}
