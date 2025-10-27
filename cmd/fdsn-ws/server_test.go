package main

import (
	"database/sql"
	"io"
	"log"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/joho/godotenv"
)

var ts *httptest.Server

func setup(t *testing.T) {
	var err error

	err = godotenv.Load("env.list")
	if err != nil {
		t.Fatal(err)
	}

	initStationXML()
	initRoutes()

	S3_BUCKET = os.Getenv("S3_BUCKET")

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

	_, err = db.Exec(`DELETE FROM fdsn.event WHERE publicid = '2015p768477' or publicid = '2015p768478' or publicid = '2015p768479'`)
	if err != nil {
		t.Log(err)
	}

	_, err = db.Exec(`INSERT INTO fdsn.event (publicid, modificationtime, origintime,
	 latitude, longitude, depth, magnitude, magnitudetype, deleted, eventtype,
	 depthtype, evaluationmethod, earthmodel, evaluationmode, evaluationstatus,
	 usedphasecount, usedstationcount, originerror, azimuthalgap, minimumdistance,
	 magnitudeuncertainty, magnitudestationcount, quakeml12event, sc3ml)
	 VALUES ('2015p768477', timestamptz '2015-10-12 08:05:01.717692+00', timestamptz '2015-10-12 08:05:01.717692+00',
	 -40.57806609, 176.3257242, 23.28125, 2.3, 'magnitudetype', false, 'volcanic long-period',
	 'depthtype', 'evaluationmethod', 'earthmodel', 'evaluationmode', 'evaluationstatus',
	 0, 0, 0, 0, 0,
	 0, 0, 'quakeml12event', 'sc3ml')`)
	if err != nil {
		t.Log(err)
	}

	_, err = db.Exec(`INSERT INTO fdsn.event (publicid, modificationtime, origintime,
	 latitude, longitude, depth, magnitude, magnitudetype, deleted, eventtype,
	 depthtype, evaluationmethod, earthmodel, evaluationmode, evaluationstatus,
	 usedphasecount, usedstationcount, originerror, azimuthalgap, minimumdistance,
	 magnitudeuncertainty, magnitudestationcount, quakeml12event, sc3ml)
	 VALUES ('2015p768478', timestamptz '2015-10-12 08:05:02.717692+00', timestamptz '2015-10-12 08:05:02.717692+00',
	 -40.57806609, -176.3257242, 23.28125, 2.3, 'magnitudetype', false, 'volcanic very-long-period',
	 'depthtype', 'evaluationmethod', 'earthmodel', 'evaluationmode', 'evaluationstatus',
	 0, 0, 0, 0, 0,
	 0, 0, 'quakeml12event', 'sc3ml')`)
	if err != nil {
		t.Log(err)
	}

	_, err = db.Exec(`INSERT INTO fdsn.event (publicid, modificationtime, origintime,
	latitude, longitude, depth, magnitude, magnitudetype, deleted, eventtype,
	depthtype, evaluationmethod, earthmodel, evaluationmode, evaluationstatus,
	usedphasecount, usedstationcount, originerror, azimuthalgap, minimumdistance,
	magnitudeuncertainty, magnitudestationcount, quakeml12event, sc3ml)
	VALUES ('2015p768479', timestamptz '2015-10-12 09:05:02.717692+00', timestamptz '2015-10-12 09:05:02.717692+00',
	-23.57806609, 179.3257242, 33.28125, 2.3, 'magnitudetype', false, 'other event',
	'depthtype', 'evaluationmethod', 'earthmodel', 'evaluationmode', 'evaluationstatus',
	0, 0, 0, 0, 0,
	0, 0, 'quakeml12event', 'sc3ml')`)
	if err != nil {
		t.Log(err)
	}

	ts = httptest.NewServer(mux)

	// Silence the logging unless running with
	// go test -v
	if !testing.Verbose() {
		log.SetOutput(io.Discard)
	}
}

func teardown() {
	ts.Close()
	_ = db.Close()
}
