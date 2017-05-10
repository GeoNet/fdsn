package main

/*
slink-ws connects to a SEEDLink server and saves records to a postgres DB.
*/

import (
	_ "github.com/lib/pq"
	"log"
	"time"

	"database/sql"
	"github.com/GeoNet/collect/mseed"
	"github.com/GeoNet/collect/slink"
	"github.com/GeoNet/fdsn/internal/kit/cfg"
	"github.com/GeoNet/mtr/mtrapp"
	"os"
	"strings"
)

var db *sql.DB
var saveRecord *sql.Stmt

func main() {
	p, err := cfg.PostgresEnv()
	if err != nil {
		log.Fatalf("error reading DB config from the environment vars: %s", err)
	}

	db, err = sql.Open("postgres", p.Connection())
	if err != nil {
		log.Fatalf("error with DB config: %s", err)
	}
	defer db.Close()

	db.SetMaxIdleConns(p.MaxIdle)
	db.SetMaxOpenConns(p.MaxOpen)

ping:
	for {
		err = db.Ping()
		if err != nil {
			log.Printf("problem pinging DB - is it up and contactable: %s", err.Error())
			log.Print("sleeping and waiting for DB")
			time.Sleep(time.Second * 10)
			continue ping
		}
		break ping
	}

stmt:
	for {
		saveRecord, err = db.Prepare(`INSERT INTO fdsn.record (streamPK, start_time, raw, latency)
	SELECT streamPK, $5, $6, $7
	FROM fdsn.stream
	WHERE network = $1
	AND station = $2
	AND channel = $3
	AND location = $4`)
		if err != nil {
			log.Printf("preparing statement: %s", err)
			log.Print("sleeping and trying to prepare statement again")
			time.Sleep(time.Second * 10)
			continue stmt
		}
		break stmt
	}
	defer saveRecord.Close()

	// buffered chan to allow for DB back pressure.
	// Allows ~ 10-12 minutes of records.
	process := make(chan []byte, 200000)

	// run as many consumers from process as there are connections in the DB pool.  Tune this
	// so that the process chan doesn't fill up in normal operations.
	for i := 0; i <= p.MaxOpen; i++ {
		go save(process)
	}

	// delete old data from DB
	go expire()

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
					mtrapp.MsgRx.Inc()
				default:
					mtrapp.MsgErr.Inc()
					log.Fatal("process chan full.")
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

func save(inbound chan []byte) {
	msr := mseed.NewMSRecord()
	defer mseed.FreeMSRecord(msr)

	var err error
	var r record

	for {
		select {
		case b := <-inbound:
			err = msr.Unpack(b, 512, 0, 0)
			if err != nil {
				mtrapp.MsgErr.Inc()
				log.Printf("unpacking miniSEED record: %s", err.Error())
				continue
			}

			// have to remove trailing null padding from strings for postgres UTF8.
			r = record{
				network:  strings.Trim(msr.Network(), "\x00"),
				station:  strings.Trim(msr.Station(), "\x00"),
				channel:  strings.Trim(msr.Channel(), "\x00"),
				location: strings.Trim(msr.Location(), "\x00"),
				start:    msr.Starttime(),
				latency:  time.Now().UTC().Sub(msr.Endtime()).Seconds(),
				raw:      b,
			}

			err = r.save()
			if err != nil {
				log.Printf("saving record: %s", err.Error())
				mtrapp.MsgErr.Inc()
			} else {
				mtrapp.MsgProc.Inc()
			}
		}
	}
}

func expire() {
	ticker := time.NewTicker(time.Minute).C
	var err error
	for {
		select {
		case <-ticker:
			_, err = db.Exec(`DELETE FROM fdsn.record WHERE start_time < now() - interval '48 hours'`)
			if err != nil {
				log.Printf("deleting old records: %s", err.Error())
			}
		}
	}
}
