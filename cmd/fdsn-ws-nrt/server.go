package main

import (
	"github.com/GeoNet/fdsn/internal/mseednrt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

var (
	cacheDir = os.Getenv("CACHE_DIR")
	cache    mseednrt.Cache
)

func main() {
	if cacheDir == "" {
		log.Fatal("CACHE_DIR is not set")
	}

	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		log.Fatal("cache dir", cacheDir, "doesn't exists")
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

	log.Printf("creating record cache size %d bytes", cacheSize)

	cache = mseednrt.InitCache("TestCache_List", 1000000, 10000, time.Second*10, cacheDir)

	log.Println("starting server")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
