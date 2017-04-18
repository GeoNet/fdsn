package main

/*
Loads miniSEED key listings into the fdsn-ws-event holdings via a PUT.  The fdsn server
needs to be running and the keys need to match.

Use the aws cli to create a listing of keys to be loaded.  One per line.
*/

import (
	"encoding/csv"
	"log"
	"net/http"
	"os"
	"sync"
)

var (
	key    = os.Getenv("FDSN_KEY")
	path   = os.Getenv("FDSN_SC3ML_URL")
	file   = os.Getenv("KEYS_FILE")
	client = &http.Client{}
)

func main() {
	http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost = 100

	keys := make(chan string)

	f, err := os.Open(file)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = 1

	input, err := r.ReadAll()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		defer close(keys)

		for _, v := range input {
			if len(v) > 0 {
				keys <- v[0]
			}
		}
	}()

	var wg sync.WaitGroup
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func() {
			procKeys(keys)

			wg.Done()
		}()
	}
	wg.Wait()
}

func procKeys(keys <-chan string) {
	var req *http.Request
	var res *http.Response
	var err error

	for k := range keys {
		if req, err = http.NewRequest("PUT", path+"/"+k, nil); err != nil {
			log.Panic(err)
		}

		req.SetBasicAuth("", key)

		if res, err = client.Do(req); err != nil {
			log.Printf("ERROR: put for %s: %s", k, err)
		}
		if err != nil {
			log.Print(err)
		}

		if res != nil {
			if res.StatusCode != http.StatusOK {
				log.Printf("ERROR: non 200 response (%d) for  post for %s", res.StatusCode, k)
			}

			res.Body.Close()
		} else {
			log.Printf("nil response body for %s", k)
		}

	}
}
