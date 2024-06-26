package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/GeoNet/kit/cfg"
	"github.com/gorilla/schema"
	_ "github.com/lib/pq"
)

var (
	db        *sql.DB
	decoder   = newDecoder() // decoder for URL queries.
	S3_BUCKET string         // the S3 bucket storing the miniseed files used by dataselect
	LOG_EXTRA bool           // Whether POST body is logged.
)

var stationVersion = "1.1"
var eventVersion = "1.2"
var dataselectVersion = "1.1"
var zeroDateTime = time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)

func newDecoder() *schema.Decoder {
	decoder := schema.NewDecoder()
	// Handle comma separated parameters (eg: net, sta, loc, cha, etc)
	decoder.RegisterConverter([]string{}, func(input string) reflect.Value {
		return reflect.ValueOf(strings.Split(input, ","))
	})
	return decoder
}

func main() {
	var err error
	if S3_BUCKET = os.Getenv("S3_BUCKET"); S3_BUCKET == "" {
		log.Fatal("ERROR: S3_BUCKET environment variable is not set")
	}

	LOG_EXTRA = false
	if log_extra := os.Getenv("LOG_EXTRA"); log_extra == "true" {
		LOG_EXTRA = true
	}

	p, err := cfg.PostgresEnv()
	if err != nil {
		log.Fatalf("error reading DB config from the environment vars: %s", err)
	}

	// set a statement timeout to cancel any very long running DB queries.
	// Value is int milliseconds.
	// https://www.postgresql.org/docs/9.5/static/runtime-config-client.html
	db, err = sql.Open("postgres", p.Connection()+" statement_timeout=600000")
	if err != nil {
		log.Fatalf("error with DB config: %s", err)
	}
	defer db.Close()

	db.SetMaxIdleConns(p.MaxIdle)
	db.SetMaxOpenConns(p.MaxOpen)

	if err = db.Ping(); err != nil {
		log.Println("ERROR: problem pinging DB - is it up and contactable? 500s will be served")
	}

	initDataselectTemplate()
	initEventTemplate()
	initStationTemplate()
	initStationXML()
	initRoutes()

	setupStationXMLUpdater()

	log.Println("starting server")
	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  1 * time.Minute,
		WriteTimeout: 10 * time.Minute,
	}
	log.Fatal(server.ListenAndServe())
}
