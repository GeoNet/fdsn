package main

/*
slink-ws connects to a SEEDLink server and saves records to a postgres DB.
*/

import (
	"log"
	"os"
	"time"

	"github.com/GeoNet/kit/metrics"
	"github.com/GeoNet/kit/seis/sl"
	_ "github.com/lib/pq"
)

const maxPatchBefore = 10 * time.Minute

var server = os.Getenv("SLINK_HOST")
var netto = 60 * time.Second
var keepalive = 1 * time.Second
var streams = "*_*"

func main() {
	var a app

	err := a.initDB()
	if err != nil {
		log.Fatal(err)
	}
	defer a.close()

	// buffered chan to allow for DB back pressure.
	// Allows ~ 10-12 minutes of records.
	process := make(chan []byte, 200000)

	/// run as many consumers for process as there are connections in the DB pool.
	for i := 0; i <= a.maxOpen; i++ {
		go a.save(process)
	}

	go a.expire()

	// Request old data
	latest, err := a.latestTS()
	if err != nil || time.Since(latest) > maxPatchBefore {
		// Limit number of missing data to start from "maxPatchBefore ago" if we've missed too much
		latest = time.Now().UTC().Add(-1 * maxPatchBefore)
	}

	log.Println("listening for packets from seedlink")

	// additional logic in recv loop handles cases where the connection to
	// SEEDLink is hung or a corrupt packet is received.  In these
	// cases the program exits and the service should restart it.
	for {
		if latest, err = a.latestTS(); err != nil || time.Since(latest) > maxPatchBefore {
			// In fact, whenever we can't get the latest it means database is not working properly.
			// We would facing error when doing save()
			latest = time.Now().UTC().Add(-1 * maxPatchBefore)
		}
		slink := sl.NewSLink(
			sl.SetServer(server),
			sl.SetNetTo(netto),
			sl.SetKeepAlive(keepalive),
			sl.SetStart(latest),
			sl.SetStreams(streams),
		)
		if err := slink.Collect(func(seq string, data []byte) (bool, error) {
			select {
			case process <- data:
				metrics.MsgRx()
			default:
				log.Fatal("process chan full, exiting")
			}
			return false, nil
		}); err != nil {
			log.Println("slink.Collect:", err)
		}
	}
}
