package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"github.com/GeoNet/kit/weft"
	"net/http"
	"net/http/httputil"
)

var mux *http.ServeMux

func init() {
	mux = http.NewServeMux()

	// station and event services are proxied
	stationDirector := func(r *http.Request) {
		r.Host = "service.geonet.org.nz"
		r.URL.Scheme = "https"
		r.URL.Host = "service.geonet.org.nz"
	}

	eventDirector := func(r *http.Request) {
		r.Host = "service.geonet.org.nz"
		r.URL.Scheme = "https"
		r.URL.Host = "service.geonet.org.nz"
	}

	mux.HandleFunc("/", weft.MakeHandler(weft.NoMatch, weft.TextError))
	mux.HandleFunc("/soh/up", weft.MakeHandler(weft.Up, weft.TextError))
	mux.HandleFunc("/soh", weft.MakeHandler(soh, weft.UseError))

	mux.Handle("/fdsnws/station/", &httputil.ReverseProxy{Director: stationDirector})
	mux.Handle("/fdsnws/event/", &httputil.ReverseProxy{Director: eventDirector})

	// This service implements the dataselect spec from http://www.fdsn.org/webservices/FDSN-WS-Specifications-1.1.pdf.
	mux.HandleFunc("/fdsnws/dataselect/1", weft.MakeHandler(fdsnDataselectV1Index, weft.TextError))
	mux.HandleFunc("/fdsnws/dataselect/1/query", weft.MakeDirectHandler(fdsnDataselectV1Handler, weft.TextError))
	mux.HandleFunc("/fdsnws/dataselect/1/version", weft.MakeHandler(fdsnDataselectVersion, weft.TextError))
	mux.HandleFunc("/fdsnws/dataselect/1/application.wadl", weft.MakeHandler(fdsnDataselectWadl, weft.TextError))
}

func soh(r *http.Request, h http.Header, b *bytes.Buffer) error {
	err := weft.CheckQuery(r, []string{"GET"}, []string{}, []string{})
	if err != nil {
		return err
	}

	// miniSEED records arrive continuously.  There should be records in the DB in the last hour.
	var numRecords sql.NullInt64
	if err != nil {
		return err
	}

	if numRecords.Int64 == 0 {
		b.Write([]byte("<html><head></head><body>have zero miniSEED records for the last hour.</body></html>"))
		return weft.StatusError{Code: http.StatusServiceUnavailable}
	}

	b.Write([]byte(fmt.Sprintf("<html><head></head><body>have %d miniSEED records for the last hour.</body></html>", numRecords.Int64)))

	return nil
}
