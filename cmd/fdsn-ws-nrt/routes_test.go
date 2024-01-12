package main

import (
	"database/sql"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	wt "github.com/GeoNet/kit/weft/wefttest"
)

var testServer *httptest.Server

var routes = wt.Requests{
	{ID: wt.L(), URL: "/soh"},
	// fdsn-ws-dataselect
	{ID: wt.L(), URL: "/fdsnws/dataselect/1", Content: "text/html"},
	{ID: wt.L(), URL: "/fdsnws/dataselect/1/version", Content: "text/plain"},
	{ID: wt.L(), URL: "/fdsnws/dataselect/1/application.wadl", Content: "application/xml"},
	// valid record (inserted by data_nrt_test.go)
	// nrt service only return latest 7 days
	{ID: wt.L(), URL: "/fdsnws/dataselect/1/query?starttime=2016-03-19T00:00:00&endtime=2016-03-19T01:00:00&network=NZ&station=ABAZ&location=10&channel=EHE",
		Status: http.StatusNoContent},
	// an invalid network or no files matching query should give 404 (could also give 204 as per spec)
	// Note: though the response is 204 but the mseed header already set before the content checking.
	// nrt service only return latest 7 days
	{ID: wt.L(), URL: "/fdsnws/dataselect/1/query?starttime=2016-01-09T00:00:00&endtime=2016-01-09T23:00:00&network=INVALID_NETWORK&station=CHST&location=01&channel=LOG",
		Status: http.StatusNoContent},
	// very old time range, no files:
	{ID: wt.L(), URL: "/fdsnws/dataselect/1/query?starttime=1900-01-09T00:00:00&endtime=1900-01-09T01:00:00&network=NZ&station=CHST&location=01&channel=LOG",
		Status: http.StatusNoContent},
	// spam
	{ID: wt.L(), URL: "/fdsnws/dataselect/1/query?channel=LOG&endtime=1900-01-09T01:00:00&location=01&network=c:/Windows/system.ini&starttime=1900-01-09T00:00:00&station=CHST",
		Status: http.StatusBadRequest},
	{ID: wt.L(), URL: "/fdsnws/dataselect/1/query?channel=LOG&endtime=1900-01-09T01%3A00%3A00&location=01&network=NZ&starttime=1900-01-09T00%3A00%3A00&station=c%3A%2FWindows%2Fsystem.ini",
		Status: http.StatusBadRequest},
	{ID: wt.L(), URL: "/fdsnws/dataselect/1/query?channel=LOG&endtime=1900-01-09T01%3A00%3A00&location=01&network=c%3A%2FWindows%2Fsystem.ini&starttime=1900-01-09T00%3A00%3A00&station=CHST",
		Status: http.StatusBadRequest},
	{ID: wt.L(), URL: "/fdsnws/dataselect/1/query?channel=LOG&endtime=c%3A%2FWindows%2Fsystem.ini&location=01&network=*&starttime=1900-01-09T00%3A00%3A00&station=CHST",
		Status: http.StatusBadRequest},
	{ID: wt.L(), URL: "/fdsnws/dataselect/1/query?unknownparam=1",
		Status: http.StatusBadRequest},
	// post
	// nrt service only return latest 7 days
	{ID: wt.L(), Method: "POST", URL: "/fdsnws/dataselect/1/query", PostBody: []byte("NZ ABAZ 10 EHE 2016-03-19T00:00:00 2016-03-19T01:00:00"),
		Status: http.StatusNoContent},
}

func TestRoutes(t *testing.T) {
	setup(t)
	defer teardown()

	for _, r := range routes {
		if b, err := r.Do(testServer.URL); err != nil {
			t.Error(err)
			t.Error(string(b))
		}
	}
}

func setup(t *testing.T) {
	var err error
	db, err = sql.Open("postgres", "host=localhost connect_timeout=300 user=fdsn_r password=test dbname=fdsn sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}

	err = db.Ping()

	if err != nil {
		log.Fatal(err)
	}

	testServer = httptest.NewServer(mux)

	// Silence the logging unless running with
	// go test -v
	if !testing.Verbose() {
		log.SetOutput(io.Discard)
	}
}

func teardown() {
	testServer.Close()
}
