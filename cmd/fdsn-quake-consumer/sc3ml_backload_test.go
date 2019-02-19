// +build backload

package main

import (
	"database/sql"
	"github.com/GeoNet/kit/cfg"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"
	"testing"
)

// TestBackload can be used to load SC3ML into the FDSN database.  Files are read
// from /work/seiscompml07-to-load

// Compile and keep the test binary.  This binary can be copied to other systems.
//
// go test -c -tags=backload
// ./fdsn-quake-consumer.test -test.v -test.timeout 2h -test.run ^TestBackload$
func TestBackload(t *testing.T) {
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

	var dir = "/work/seiscompml07-to-load"

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	log.Printf("found %d files\n", len(files))

	sc3ml := make(chan string)

	go func() {
		defer close(sc3ml)

		for _, f := range files {
			if !strings.HasSuffix(f.Name(), ".xml") {
				continue
			}
			sc3ml <- dir + "/" + f.Name()
		}
	}()

	var wg sync.WaitGroup
	wg.Add(40)

	for i := 0; i < 40; i++ {
		go func(in <-chan string) {
			var e event
			var b []byte
			var err error
			var f string

			defer wg.Done()

			for f = range in {
				b, err = ioutil.ReadFile(f)
				if err != nil {
					log.Println(err)
					continue
				}

				e = event{}

				err = unmarshal(b, &e)
				if err != nil {
					log.Println(err)
					continue
				}

				err = e.save()
				if err != nil {
					log.Println(err)
					continue
				}

				err = os.Remove(f)
				if err != nil {
					log.Println(err)
				}
			}
		}(sc3ml)
	}

	wg.Wait()

}
