package main

import (
	"database/sql"
	"io/ioutil"
	"log"
	"net/http/httptest"
	"os"
	"testing"
)

var ts *httptest.Server

func setup(t *testing.T) {
	var err error

	S3_BUCKET = os.Getenv("S3_BUCKET")
	MAX_WORKERS = 10

	// need a db write user for adding test data.
	// should use a db r/o user in prod.
	db, err = sql.Open("postgres", "host=localhost connect_timeout=300 user=fdsn_w password=test dbname=fdsn sslmode=disable statement_timeout=600000")
	if err != nil {
		t.Fatalf("ERROR: problem with DB config: %s", err)
	}

	db.SetMaxIdleConns(5)
	db.SetMaxOpenConns(15)

	if err = db.Ping(); err != nil {
		t.Fatal("ERROR: problem pinging DB")
	}

	ts = httptest.NewServer(mux)

	// Silence the logging unless running with
	// go test -v
	if !testing.Verbose() {
		log.SetOutput(ioutil.Discard)
	}
}

func teardown() {
	ts.Close()
	db.Close()
}
