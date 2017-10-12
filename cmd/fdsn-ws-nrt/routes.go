package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"github.com/GeoNet/fdsn/internal/weft"
	"log"
	"net/http"
	"net/http/httputil"
)

var mux *http.ServeMux

func init() {
	mux = http.NewServeMux()

	// station and event services are proxied
	stationDirector := func(r *http.Request) {
		r.Host = "beta-service.geonet.org.nz"
		r.URL.Scheme = "http"
		r.URL.Host = "beta-service.geonet.org.nz"
	}

	eventDirector := func(r *http.Request) {
		r.Host = "beta-service.geonet.org.nz"
		r.URL.Scheme = "http"
		r.URL.Host = "beta-service.geonet.org.nz"
	}

	mux.Handle("/fdsnws/station/", &httputil.ReverseProxy{Director: stationDirector})
	mux.Handle("/fdsnws/event/", &httputil.ReverseProxy{Director: eventDirector})

	// This service implements the dataselect spec from http://www.fdsn.org/webservices/FDSN-WS-Specifications-1.1.pdf.
	mux.HandleFunc("/fdsnws/dataselect/1", weft.MakeHandlerAPI(fdsnDataselectV1Index))
	mux.HandleFunc("/fdsnws/dataselect/1/query", weft.MakeSimpleHandler(fdsnDataselectV1Handler))
	mux.HandleFunc("/fdsnws/dataselect/1/version", weft.MakeHandlerAPI(fdsnDataselectVersion))
	mux.HandleFunc("/fdsnws/dataselect/1/application.wadl", weft.MakeHandlerAPI(fdsnDataselectWadl))

	mux.HandleFunc("/", weft.MakeHandlerAPI(noMatch))

	// routes for balancers and probes.
	mux.HandleFunc("/soh/up", http.HandlerFunc(up))
	mux.HandleFunc("/soh", http.HandlerFunc(soh))
}

func noMatch(r *http.Request, h http.Header, b *bytes.Buffer) *weft.Result {
	if res := weft.CheckQuery(r, []string{}, []string{}); !res.Ok {
		return res
	}

	return &weft.NotFound
}

// up is for testing that the app has started e.g., for with load balancers.
// It indicates the app is started.  It may still be serving errors.
// Not useful for inclusion in app metrics so weft not used.
func up(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if res := weft.CheckQuery(r, []string{}, []string{}); !res.Ok {
		w.Header().Set("Surrogate-Control", "max-age=86400")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.Header().Set("Surrogate-Control", "max-age=10")

	w.Write([]byte("<html><head></head><body>up</body></html>"))
}

// soh is for external service probes.
// writes a service unavailable error to w if the service is not working.
// Not useful for inclusion in app metrics so weft not used.
func soh(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if res := weft.CheckQuery(r, []string{}, []string{}); !res.Ok {
		w.Header().Set("Surrogate-Control", "max-age=86400")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// miniSEED records arrive continuously.  There should be records in the DB in the last hour.
	var numRecords sql.NullInt64
	err := db.QueryRow(`SELECT count(*) FROM fdsn.record WHERE start_time > now() - interval '1 hour'`).Scan(&numRecords)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("<html><head></head><body>service error</body></html>"))
		log.Printf("ERROR: soh service error %s", err)
		return
	}

	if numRecords.Int64 == 0 {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("<html><head></head><body>have zero miniSEED records for the last hour.</body></html>"))
		return
	}

	w.Header().Set("Surrogate-Control", "max-age=10")

	w.Write([]byte(fmt.Sprintf("<html><head></head><body>have %d miniSEED records for the last hour.</body></html>", numRecords.Int64)))
}
