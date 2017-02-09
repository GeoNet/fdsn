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

	db, err = sql.Open("postgres", "host=localhost connect_timeout=300 user=fdsn_w password=test dbname=fdsn sslmode=disable")
	if err != nil {
		t.Fatalf("ERROR: problem with DB config: %s", err)
	}

	db.SetMaxIdleConns(5)
	db.SetMaxOpenConns(15)

	if err = db.Ping(); err != nil {
		t.Fatal("ERROR: problem pinging DB")
	}

	var f *os.File
	var b []byte

	// save an event to the DB.
	if f, err = os.Open("etc/2015p768477.xml"); err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if b, err = ioutil.ReadAll(f); err != nil {
		t.Fatal(err)
	}

	var e event

	if err = unmarshal(b, &e); err != nil {
		t.Error(err)
	}

	if err = e.save(); err != nil {
		t.Error(err)
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
