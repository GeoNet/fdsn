package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/GeoNet/kit/cfg"
	"github.com/GeoNet/kit/metrics"
	ms "github.com/GeoNet/kit/seis/ms"
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
		return fmt.Errorf("error reading DB config from the environment vars: %w", err)
	}

	a.db, err = sql.Open("postgres", p.Connection())
	if err != nil {
		return fmt.Errorf("error with DB config: %w", err)
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
		return fmt.Errorf("error preparing saveRecord stmt: %w", err)
	}

	return nil
}

func (a *app) save(inbound chan []byte) {
	for b := range inbound {
		t := metrics.Start()
		msr, err := ms.NewRecord(b)
		if err != nil {
			metrics.MsgErr()
			log.Printf("unpacking miniSEED record: %s", err.Error())
			continue
		}

		for {
			err = a.saveRecord(record{
				network:      msr.Network(),
				station:      msr.Station(),
				channel:      msr.Channel(),
				location:     msr.Location(),
				start:        msr.StartTime(),
				latency_tx:   time.Now().UTC().Sub(msr.EndTime()).Seconds(),
				latency_data: time.Now().UTC().Sub(msr.StartTime()).Seconds(),
				raw:          b,
			})
			if err != nil {
				metrics.MsgErr()
				log.Printf("error saving record sleeping and trying again: %s", err)
				time.Sleep(time.Second * 10)
				continue
			}
			break
		}

		if err := t.Track("save"); err != nil {
			log.Print(err)
		}
		metrics.MsgProc()

	}
}

// expire removes old data from the DB.  The archive runs 7 days between real time.  Keep
// 8 days to allow some overlap.
func (a *app) expire() {
	ticker := time.NewTicker(time.Minute).C
	var err error
	for range ticker {
		_, err = a.db.Exec(`DELETE FROM fdsn.record WHERE start_time < now() - interval '8 days'`)
		if err != nil {
			log.Printf("deleting old records: %s", err.Error())
		}
	}
}

// returns the latest records's time
func (a *app) latestTS() (t time.Time, err error) {
	var ts string
	err = a.db.QueryRow(`SELECT start_time FROM fdsn.record ORDER BY start_time DESC LIMIT 1`).Scan(&ts)
	if err != nil {
		return
	}

	err = t.UnmarshalText([]byte(ts))
	return
}

func (a *app) close() {
	a.saveRecordStmt.Close()
	a.db.Close()
}
