package main

import (
	"database/sql"
	_ "github.com/GeoNet/fdsn/internal/ddoghttp"
	"github.com/GeoNet/fdsn/internal/platform/cfg"
	"github.com/GeoNet/fdsn/internal/platform/metrics"
	"github.com/golang/groupcache"
	"github.com/gorilla/schema"
	_ "github.com/lib/pq"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

var (
	db          *sql.DB
	decoder     = schema.NewDecoder() // decoder for URL queries.
	Prefix      string                // prefix for logging
	recordStmt  *sql.Stmt
	recordCache *groupcache.Group
)

func init() {
	if Prefix != "" {
		log.SetPrefix(Prefix + " ")
	}
}

func main() {
	p, err := cfg.PostgresEnv()
	if err != nil {
		log.Fatalf("error reading DB config from the environment vars: %s", err)
	}

	size := os.Getenv("CACHE_SIZE")
	if size == "" {
		log.Fatal("CACHE_SIZE env var must be set")
	}

	cacheSize, err := strconv.ParseInt(size, 10, 64)
	if err != nil {
		log.Fatalf("error parsing CACHE_SIZE env var %s", err.Error())
	}

	cacheSize = cacheSize * 1000000000

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

	recordStmt, err = db.Prepare(`SELECT raw FROM fdsn.record WHERE streampk =
                                  (SELECT streampk FROM fdsn.stream WHERE network = $1 AND station = $2 AND channel = $3 AND location = $4)
	                          AND start_time = $5`)
	if err != nil {
		log.Printf("error preparing record statement %s", err.Error())
	}

	if err = db.Ping(); err != nil {
		log.Println("ERROR: problem pinging DB - is it up and contactable? 500s will be served")
	}

	log.Printf("creating record cache size %d bytes", cacheSize)

	recordCache = groupcache.NewGroup("record", cacheSize, groupcache.GetterFunc(recordGetter))

	go func() {
		ticker := time.Tick(time.Second * 30)

		for {
			select {
			case <-ticker:
				t := metrics.Start()
				err := primeCache(time.Now().UTC().Add(time.Second * -40))
				if err != nil {
					log.Printf("priming cache %s", err.Error())
				}
				t.Track("primeCache")
				log.Printf("record cache: %+v", recordCache.CacheStats(groupcache.MainCache))
			}
		}
	}()

	log.Println("starting server")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
