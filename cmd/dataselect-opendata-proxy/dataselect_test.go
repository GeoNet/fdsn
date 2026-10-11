package main

import (
	"crypto/sha256"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

const query = "/fdsnws/dataselect/1/query?network=NZ&station=AKCZ&channel=*&location=10&start=2026-04-01T22:00:00Z&end=2026-04-02T02:00:00Z"
const testDataFile = "testdata/AKCZ.D"

// TestDataselectSample starts the proxy server and sends the same
// query that was used to produce testdata/AKCZ.D, then compares the binary
func TestDataselectSample(t *testing.T) {
	if err := initS3Client(); err != nil {
		t.Fatalf("initS3Client: %v", err)
	}

	initRoutes()

	if !testing.Verbose() {
		log.SetOutput(io.Discard)
	}

	ts := httptest.NewServer(mux)
	defer ts.Close()

	resp, err := http.Get(ts.URL + query)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}

	got, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading response: %v", err)
	}

	want, err := os.ReadFile(testDataFile)
	if err != nil {
		t.Fatalf("reading testdata: %v", err)
	}

	if sha256.Sum256(got) != sha256.Sum256(want) {
		t.Fatalf("sha256 mismatch")
	}
}
