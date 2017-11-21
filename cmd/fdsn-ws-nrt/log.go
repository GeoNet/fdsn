package main

import (
	"github.com/GeoNet/kit/weft"
	"log"
	"os"
	"strings"
)

var Prefix string

func init() {
	logger := log.New(os.Stderr, "", log.LstdFlags)

	if Prefix != "" {
		log.SetPrefix(Prefix + " ")
		logger.SetPrefix(Prefix + " ")
	}

	weft.SetLogger(logger)

	// find the hostname and appname for use with metrics.
	h, _ := os.Hostname()

	a := os.Args[0]
	a = strings.Replace(a[strings.LastIndex(a, "/")+1:], "-", "_", -1)

	weft.DataDog(os.Getenv("DDOG_API_KEY"), h, a, logger)
}
