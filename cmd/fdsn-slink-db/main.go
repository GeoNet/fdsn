package main

/*
slink-ws connects to a SEEDLink server and saves records to a postgres DB.
*/

import (
	_ "github.com/GeoNet/fdsn/internal/ddogmsg"
	"github.com/GeoNet/fdsn/internal/platform/metrics"
	"github.com/GeoNet/kit/slink"
	_ "github.com/lib/pq"
	"log"
	"os"
	"time"
)

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

	// TODO request old data?

	slconn := slink.NewSLCD()
	defer slink.FreeSLCD(slconn)

	slconn.SetNetDly(30)
	slconn.SetNetTo(300)
	slconn.SetKeepAlive(0)

	slconn.SetSLAddr(os.Getenv("SLINK_HOST"))
	defer slconn.Disconnect()

	slconn.ParseStreamList("*_*", "")

	log.Println("listening for packets from seedlink")

	last := time.Now()

	// additional logic in recv loop handles cases where the connection to
	// SEEDLink is hung or a corrupt packet is received.  In these
	// cases the program exits and the service should restart it.
recv:
	for {
		if time.Now().Sub(last) > 300.0*time.Second {
			log.Print("ERROR: no packets for 300s connection may be hung, exiting")
			break recv
		}

		// collect packets, blocking connection.
		switch p, rc := slconn.Collect(); rc {
		case slink.SLTERMINATE:
			log.Println("ERROR: slink terminate signal")
			break recv
		case slink.SLNOPACKET:
			// blocking connection so should never hit this option.
			time.Sleep(5 * time.Millisecond)
			continue recv
		case slink.SLPACKET:
			if p != nil && p.PacketType() == slink.SLDATA {
				select {
				case process <- p.GetMSRecord():
					metrics.MsgRx()
				default:
					log.Fatal("process chan full, exiting")
				}
			}
			last = time.Now()
		default:
			// bad packet.  Exit and allow the service to restart.
			log.Println("ERROR: invalid packet")
			break recv
		}
	}

	log.Println("ERROR: unexpected exit")
}
