package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/GeoNet/kit/health"
	"github.com/GeoNet/kit/weft"
)

const servicePort = ":8080"

var (
	LOG_EXTRA bool
)

const dataselectVersion = "1.1"

func main() {
	if health.RunningHealthCheck() {
		healthCheck()
	}

	LOG_EXTRA = false
	if logExtra := os.Getenv("LOG_EXTRA"); logExtra == "true" {
		LOG_EXTRA = true
	}

	if err := initS3Client(); err != nil {
		log.Fatalf("error creating S3 client: %s", err)
	}

	initRoutes()

	log.Println("starting dataselect-opendata-proxy server")
	server := &http.Server{
		Addr:         servicePort,
		Handler:      mux,
		ReadTimeout:  1 * time.Minute,
		WriteTimeout: 10 * time.Minute,
	}
	log.Fatal(server.ListenAndServe())
}

func healthCheck() {
	timeout := 30 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	msg, err := health.Check(ctx, fmt.Sprintf("%s/soh", servicePort), timeout)
	if err != nil {
		log.Printf("status: %v", err)
		os.Exit(1)
	}
	log.Printf("status: %s", string(msg))
	os.Exit(0)
}

func init() {
	logger := log.New(os.Stderr, "", log.LstdFlags)
	weft.SetLogger(logger)
	weft.EnableLogRequest(true)
	weft.EnableLogPostBody(true)
}
