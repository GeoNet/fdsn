package main

import (
	"database/sql"
	"github.com/GeoNet/fdsn/internal/kit/cfg"
	"github.com/gorilla/schema"
	_ "github.com/lib/pq"
	"log"
	"net/http"
	"os"
	"time"
)

// TODO set this from config?
// Make it larger?  How many GB for a day?
const recordCacheSize = 4000000000 // RAM for nrt records groupcache.

var (
	db           *sql.DB
	decoder      = schema.NewDecoder() // decoder for URL queries.
	Prefix       string                // prefix for logging
	S3_BUCKET    string                // the S3 bucket storing the miniseed files used by dataselect
	zeroDateTime time.Time
)

func init() {
	if Prefix != "" {
		log.SetPrefix(Prefix + " ")
	}
	zeroDateTime = time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)
}

func main() {
	var err error
	if S3_BUCKET = os.Getenv("S3_BUCKET"); S3_BUCKET == "" {
		log.Fatal("ERROR: S3_BUCKET environment variable is not set")
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

	setupStationXMLUpdater()

	log.Println("starting server")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
