package main

import (
	"github.com/GeoNet/kit/metrics"
	"log"
	"os"
)

var Prefix string

func init() {
	logger := log.New(os.Stderr, "", log.LstdFlags)

	if Prefix != "" {
		log.SetPrefix(Prefix + " ")
		logger.SetPrefix(Prefix + " ")
	}

	metrics.DataDogMsg(os.Getenv("DDOG_API_KEY"), metrics.HostName(), metrics.AppName(), logger)
}
