package main

import (
	"database/sql"
	"github.com/GeoNet/fdsn/internal/kit/cfg"
	"github.com/gorilla/schema"
	_ "github.com/lib/pq"
	"log"
	"net/http"
	"os"
)

var (
	db        *sql.DB
	decoder   = schema.NewDecoder() // decoder for URL queries.
	Prefix    string                // prefix for logging
	S3_BUCKET string                // the S3 bucket storing the miniseed files used by dataselect
)

func init() {
	if Prefix != "" {
		log.SetPrefix(Prefix + " ")
	}

	if S3_BUCKET = os.Getenv("S3_BUCKET"); S3_BUCKET == "" {
		log.Fatal("ERROR: S3_BUCKET environment variable is not set")
	}
}

func main() {

	p, err := cfg.PostgresEnv()
	if err != nil {
		log.Fatalf("error reading DB config from the environment vars: %s", err)
	}

	db, err = sql.Open("postgres", p.Connection())
	if err != nil {
		log.Fatalf("error with DB config: %s", err)
	}
	defer db.Close()

	db.SetMaxIdleConns(p.MaxIdle)
	db.SetMaxOpenConns(p.MaxOpen)

	if err = db.Ping(); err != nil {
		log.Println("ERROR: problem pinging DB - is it up and contactable? 500s will be served")
	}

	log.Println("starting server")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
