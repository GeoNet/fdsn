package main

/*
shakenz-slink connects to a SEEDLink server and requests strong motion streams.
It calculates PGA and PGV for each data packet received.  If the calculated
values are over a threshold then the results are save to a DB.
*/

import (
	"log"
	"time"

	"github.com/GeoNet/collect/mseed"
	"github.com/GeoNet/collect/slink"
	"github.com/GeoNet/haz/database"
	"os"
)

var cfg config

func main() {
	var err error

	err = cfg.load()
	if err != nil {
		log.Fatalf("unable to load stream config file shakenz-slink.pb: %v", err)
	}

	// Work is in real time no point in pinging or waiting for the db before reading packets.
	db, err = database.InitPG()
	if err != nil {
		log.Fatalf("ERROR: problem with DB config: %s", err)
	}
	defer db.Close()

	slconn := slink.NewSLCD()
	defer slink.FreeSLCD(slconn)

	slconn.SetNetDly(0)
	slconn.SetNetTo(300)
	slconn.SetKeepAlive(0)

	slconn.SetSLAddr(os.Getenv("SLINK_HOST"))
	defer slconn.Disconnect()

	slconn.ParseStreamList("*_*", "BN? HN?")

	// buffered chan to allow for DB back pressure.
	process := make(chan shaking, 2500)
	go save(process)
	go expire()

	// reload config at this interval.  Could be triggered from a message
	ticker := time.NewTicker(time.Hour * 12).C

	log.Println("listening for packets from seedlink")

	msr := mseed.NewMSRecord()
	defer mseed.FreeMSRecord(msr)
	last := time.Now()

	// additional logic in loop handles cases where the connection to
	// SEEDLink is hung or a corrupt packet is received.  In these
	// cases the program exits and the service should restart it.
loop:
	for {
		if time.Now().Sub(last) > 300.0*time.Second {
			log.Print("ERROR: no packets for 300s connection may be hung, exiting")
			break loop
		}

		// block message reception to reload config.
		// avoids race conditions on packets in flight when
		// the config changes.
		select {
		case <-ticker:
			cfg.load()
		default:
			// recover and process packet
			switch p, rc := slconn.CollectNB(); rc {
			case slink.SLTERMINATE:
				log.Println("ERROR: slink terminate signal")
				break loop
			case slink.SLNOPACKET:
				time.Sleep(100 * time.Millisecond)
				continue loop
			case slink.SLPACKET:
				if p != nil && p.PacketType() == slink.SLDATA {
					if err = msr.Unpack(p.GetMSRecord(), 512, 1, 0); err != nil {
						log.Print("ERROR: error unpacking miniseed record", err.Error())
						continue
					}

					select {
					case process <- toShaking(msr):
					default:
						log.Print("WARN: process chan full dropping packets")
					}
				}
				last = time.Now()
			default:
				// bad packet.  Exit and allow the service to restart.
				log.Println("ERROR: invalid packet")
				break loop
			}
		}
	}

	log.Println("ERROR: unexpected exit")
}
