package main

import (
	"database/sql"
	wt "github.com/GeoNet/kit/weft/wefttest"
	"io/ioutil"
	"log"
	"net/http/httptest"
	"testing"
)

var testServer *httptest.Server

var routes = wt.Requests{
	{ID: wt.L(), URL: "/soh"},
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
		log.SetOutput(ioutil.Discard)
	}
}

func teardown() {
	testServer.Close()
}
