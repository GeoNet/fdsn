package main

/*
Loads S3CML into the fdsn-ws-event service via a POST.  The fdsn server
needs to be running and the keys need to match.

If a file is successfully loaded it is deleted.

Sync SC3ML from AWS using:

   aws s3 sync s3://seiscompml07 /work/seiscomp07 --exclude "*"  --include "2015p*"

The aws s3 command may need adjust to avoid the upload and error key prefixes.
*/

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
)

var (
	spoolDir = os.Getenv("SC3_SPOOL_DIR")
	key      = os.Getenv("FDSN_KEY")
	path     = os.Getenv("FDSN_SC3ML_URL")
	client   = &http.Client{}
)

func main() {
	files, err := ioutil.ReadDir(spoolDir)
	if err != nil {
		log.Fatal(err.Error())
	}

	sc3ml := make(chan os.FileInfo)

	go func() {
		defer close(sc3ml)

		for _, fi := range files {
			if !strings.HasSuffix(fi.Name(), ".xml") {
				continue
			}
			sc3ml <- fi
		}
	}()

	var wg sync.WaitGroup
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func() {
			procSC3ML(sc3ml)

			wg.Done()
		}()
	}
	wg.Wait()
}

func procSC3ML(sc3ml <-chan os.FileInfo) {
	for fi := range sc3ml {
		b, err := ioutil.ReadFile(spoolDir + "/" + fi.Name())
		if err != nil {
			log.Printf("WARN reading %s", fi.Name())
			continue
		}

		var buf bytes.Buffer
		buf.Write(b)

		var req *http.Request
		var res *http.Response

		if req, err = http.NewRequest("POST", path, &buf); err != nil {
			log.Panic(err)
		}

		req.SetBasicAuth("", key)

		if res, err = client.Do(req); err != nil {
			log.Printf("ERROR: post for %s: %s", fi.Name(), err)
		}

		if res.StatusCode == http.StatusOK {
			if err := os.Remove(spoolDir + "/" + fi.Name()); err != nil {
				log.Print(err)
			}
		} else {
			log.Printf("ERROR: non 200 response (%d) for  post for %s", res.StatusCode, fi.Name())
		}

		res.Body.Close()
	}
}
