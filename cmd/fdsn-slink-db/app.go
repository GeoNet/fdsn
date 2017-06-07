package main

import (
	"database/sql"
	"github.com/GeoNet/fdsn/internal/kit/cfg"
	"github.com/GeoNet/kit/mseed"
	"github.com/GeoNet/mtr/mtrapp"
	"github.com/pkg/errors"
	"log"
	"strings"
	"time"
)

// app is for shared application resources
type app struct {
	db             *sql.DB
	maxOpen        int
	saveRecordStmt *sql.Stmt
}

func (a *app) initDB() error {
	p, err := cfg.PostgresEnv()
	if err != nil {
		return errors.Wrap(err, "error reading DB config from the environment vars")
	}

	a.db, err = sql.Open("postgres", p.Connection())
	if err != nil {
		return errors.Wrap(err, "error with DB config")
	}

	a.db.SetMaxIdleConns(p.MaxIdle)
	a.db.SetMaxOpenConns(p.MaxOpen)
	a.maxOpen = p.MaxOpen

	for {
		err = a.db.Ping()
		if err != nil {
			log.Printf("error pinging a.db, waiting and trying again: %s", err.Error())
			time.Sleep(time.Second * 20)
			continue
		}
		break
	}

	a.saveRecordStmt, err = a.db.Prepare(`INSERT INTO fdsn.record (streamPK, start_time, raw, latency_tx, latency_data)
	SELECT streamPK, $5, $6, $7, $8
	FROM fdsn.stream
	WHERE network = $1
	AND station = $2
	AND channel = $3
	AND location = $4`)
	if err != nil {
		return errors.Wrap(err, "preparing saveRecord stmt")
	}

	return nil
}

func (a *app) save(inbound chan []byte) {
	msr := mseed.NewMSRecord()
	defer mseed.FreeMSRecord(msr)

	var err error

	for {
		select {
		case b := <-inbound:

			t := mtrapp.Start()

			err = msr.Unpack(b, 512, 0, 0)
			if err != nil {
				mtrapp.MsgErr.Inc()
				log.Printf("unpacking miniSEED record: %s", err.Error())
				continue
			}

			for {
				err = a.saveRecord(record{
					network:      strings.Trim(msr.Network(), "\x00"),
					station:      strings.Trim(msr.Station(), "\x00"),
					channel:      strings.Trim(msr.Channel(), "\x00"),
					location:     strings.Trim(msr.Location(), "\x00"),
					start:        msr.Starttime(),
					latency_tx:   time.Now().UTC().Sub(msr.Endtime()).Seconds(),
					latency_data: time.Now().UTC().Sub(msr.Starttime()).Seconds(),
					raw:          b,
				})
				if err != nil {
					log.Printf("error saving record sleeping and trying again: %s", err)
					time.Sleep(time.Second * 10)
					continue
				}
				break
			}

			t.Track("save")
			mtrapp.MsgProc.Inc()
		}
	}
}

func (a *app) expire() {
	ticker := time.NewTicker(time.Minute).C
	var err error
	for {
		select {
		case <-ticker:
			_, err = a.db.Exec(`DELETE FROM fdsn.record WHERE start_time < now() - interval '72 hours'`)
			if err != nil {
				log.Printf("deleting old records: %s", err.Error())
			}
		}
	}
}

func (a *app) close() {
	a.saveRecordStmt.Close()
	a.db.Close()
}
