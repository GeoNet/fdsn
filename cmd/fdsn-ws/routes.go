package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"github.com/GeoNet/kit/weft"
	"net/http"
)

var mux *http.ServeMux

func init() {
	mux = http.NewServeMux()

	mux.HandleFunc("/", weft.MakeHandler(weft.NoMatch, weft.TextError))
	mux.HandleFunc("/soh/up", weft.MakeHandler(weft.Up, weft.TextError))
	mux.HandleFunc("/soh", weft.MakeHandler(soh, weft.UseError))

	// fdsn-ws-event
	mux.HandleFunc("/fdsnws/event/1", weft.MakeHandler(fdsnEventV1Index, weft.TextError))
	mux.HandleFunc("/fdsnws/event/1/query", weft.MakeHandler(fdsnEventV1Handler, weft.TextError))
	mux.HandleFunc("/fdsnws/event/1/version", weft.MakeHandler(fdsnEventVersion, weft.TextError))
	mux.HandleFunc("/fdsnws/event/1/catalogs", weft.MakeHandler(fdsnEventCatalogs, weft.TextError))
	mux.HandleFunc("/fdsnws/event/1/contributors", weft.MakeHandler(fdsnEventContributors, weft.TextError))
	mux.HandleFunc("/fdsnws/event/1/application.wadl", weft.MakeHandler(fdsnEventWadl, weft.TextError))

	// fdsn-ws-station
	mux.HandleFunc("/fdsnws/station/1", weft.MakeHandler(fdsnStationV1Index, weft.TextError))
	mux.HandleFunc("/fdsnws/station/1/query", weft.MakeHandler(fdsnStationV1Handler, weft.TextError))
	mux.HandleFunc("/fdsnws/station/1/version", weft.MakeHandler(fdsnStationVersion, weft.TextError))
	mux.HandleFunc("/fdsnws/station/1/application.wadl", weft.MakeHandler(fdsnStationWadl, weft.TextError))

	// This service implements the dataselect spec from http://www.fdsn.org/webservices/FDSN-WS-Specifications-1.1.pdf.
	mux.HandleFunc("/fdsnws/dataselect/1", weft.MakeHandler(fdsnDataselectV1Index, weft.TextError))
	mux.HandleFunc("/fdsnws/dataselect/1/query", weft.MakeDirectHandler(fdsnDataselectV1Handler, weft.TextError))
	mux.HandleFunc("/fdsnws/dataselect/1/version", weft.MakeHandler(fdsnDataselectVersion, weft.TextError))
	mux.HandleFunc("/fdsnws/dataselect/1/application.wadl", weft.MakeHandler(fdsnDataselectWadl, weft.TextError))

	mux.HandleFunc("/metrics/fdsnws/dataselect/1/query", weft.MakeHandler(fdsnDataMetricsV1Handler, weft.TextError))

	mux.HandleFunc("/sc3ml", weft.MakeHandler(s3ml, weft.TextError))
}

// soh is for external service probes.
// writes a service unavailable error to w if the service is not working.
func soh(r *http.Request, h http.Header, b *bytes.Buffer) error {
	err := weft.CheckQuery(r, []string{"GET"}, []string{}, []string{})
	if err != nil {
		return err
	}

	// miniSEED is loaded into the archive 7 days behind real time.  There should be data in the
	// holdings DB within the last 10 days.
	var numSamples sql.NullInt64
	err = db.QueryRow(`SELECT sum(numsamples) FROM fdsn.holdings WHERE start_time > now() - interval '10 days'`).Scan(&numSamples)
	if err != nil {
		b.Write([]byte("<html><head></head><body>service error</body></html>"))
		return weft.StatusError{Code: http.StatusServiceUnavailable}
	}

	if numSamples.Int64 == 0 {
		b.Write([]byte("<html><head></head><body>holdings database has zero samples for the last ten days.</body></html>"))
		return weft.StatusError{Code: http.StatusServiceUnavailable}
	}

	_, err = b.Write([]byte(fmt.Sprintf("<html><head></head><body>holdings database has %d samples for the last ten days.</body></html>", numSamples.Int64)))

	return err
}
