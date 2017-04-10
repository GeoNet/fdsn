package main

import (
	"fmt"
	"github.com/GeoNet/collect/mseed"
	"github.com/GeoNet/delta"
	"github.com/GeoNet/haz/database"
	"github.com/GeoNet/mtr/mtrapp"
	"log"
	"time"
)

const old = time.Hour * -1
const notSending = time.Minute * -61

var db database.DB

type shaking struct {
	delta.ShakeNZStream
	pga, pgv float64
	t        time.Time
	err      error
}

type value struct {
	t time.Time
	v float64
}

type filter struct {
	m map[string]*value
}

// max returns true if v is the highest value seen for id within time old.
func (f *filter) max(id string, t time.Time, v float64) bool {
	if f.m[id] == nil || f.m[id].t.Before(time.Now().UTC().Add(old)) {
		f.m[id] = &value{t: t, v: v}
		return true
	}

	if v > f.m[id].v {
		f.m[id] = &value{t: t, v: v}
		return true
	}

	return false
}

func newFilter() filter {
	return filter{m: make(map[string]*value)}
}

// save uses filtering to reduce the number of times
// values are written to the db.
func save(inbound chan shaking) {
	pgaVerticalFilter := newFilter()
	pgaHorizontalFilter := newFilter()
	pgvVerticalFilter := newFilter()
	pgvHorizontalFilter := newFilter()

	for {
		select {
		case s := <-inbound:
			mtrapp.MsgRx.Inc()

			// set minimum values of not felt (https://en.wikipedia.org/wiki/Peak_ground_acceleration)
			// to reduce db usage.  At least one value will still go through the filters every 60 mins.
			if s.pga < 0.0017 {
				s.pga = 0.0017
			}

			if s.pgv < 0.1 {
				s.pgv = 0.1
			}

			switch {
			case s.err != nil:
				continue
			case s.t.Before(time.Now().UTC().Add(old)):
				continue
			case s.Vertical:
				if pgaVerticalFilter.max(s.Source, s.t, s.pga) {
					s.pgaVerticalSave()
				}
				if pgvVerticalFilter.max(s.Source, s.t, s.pgv) {
					s.pgvVerticalSave()
				}
			case s.Horizontal:
				if pgaHorizontalFilter.max(s.Source, s.t, s.pga) {
					s.pgaHorizontalSave()
				}
				if pgvHorizontalFilter.max(s.Source, s.t, s.pgv) {
					s.pgvHorizontalSave()
				}
			}

			if s.err != nil {
				mtrapp.MsgErr.Inc()
				log.Print(s.err)
			}
		}
	}
}

func (s *shaking) pgaVerticalSave() {
	if s.err != nil {
		return
	}

	t := mtrapp.Start()

	if s.err = db.ZeroPGAVertical(s.Source, time.Now().UTC().Add(old)); s.err != nil {
		return
	}

	s.err = db.SavePGAVertical(s.Source, s.t, s.pga)

	t.Track("pgaVerticalSave")
	log.Printf("pgaVerticalSave %s %f (%d ms)", s.Source, s.pga, t.Taken())
}

func (s *shaking) pgvVerticalSave() {
	if s.err != nil {
		return
	}

	t := mtrapp.Start()

	if s.err = db.ZeroPGVVertical(s.Source, time.Now().UTC().Add(old)); s.err != nil {
		return
	}

	s.err = db.SavePGVVertical(s.Source, s.t, s.pgv)

	t.Track("pgvVerticalSave")
	log.Printf("pgvVerticalSave %s %f (%d ms)", s.Source, s.pgv, t.Taken())
}

func (s *shaking) pgaHorizontalSave() {
	if s.err != nil {
		return
	}

	t := mtrapp.Start()

	if s.err = db.ZeroPGAHorizontal(s.Source, time.Now().UTC().Add(old)); s.err != nil {
		return
	}

	s.err = db.SavePGAHorizontal(s.Source, s.t, s.pga)

	t.Track("pgaHorizontalSave")
	log.Printf("pgaHorizontalSave %s %f (%d ms)", s.Source, s.pga, t.Taken())
}

func (s *shaking) pgvHorizontalSave() {
	if s.err != nil {
		return
	}

	t := mtrapp.Start()

	if s.err = db.ZeroPGVHorizontal(s.Source, time.Now().UTC().Add(old)); s.err != nil {
		return
	}

	s.err = db.SavePGVHorizontal(s.Source, s.t, s.pgv)

	t.Track("pgvHorizontalSave")
	log.Printf("pgvHorizontalSave %s %f (%d ms)", s.Source, s.pgv, t.Taken())
}

/*
toShaking converts an mseed.MSRecord to shaking.
s.err will be set if m is to old, not suitable for
calculating shaking, or config is not available for
the stream.
*/
func toShaking(m *mseed.MSRecord) (s shaking) {
	if m == nil {
		s.err = fmt.Errorf("nil pointer m")
		return
	}

	cfg.RLock()
	stream, ok := cfg.m[m.SrcName(0)]
	cfg.RUnlock()
	if !ok {
		s.err = fmt.Errorf("%s: no stream config", m.SrcName(0))
		return
	}

	s.ShakeNZStream = stream.ShakeNZStream

	if stream.SampleRate != float64(m.Samprate()) {
		// filters are defined for an expected sample rate.
		s.err = fmt.Errorf("%s: sample rate missmatch rx=%f cfg=%f", s.StreamId, m.Samprate(), stream.SampleRate)
		return
	}

	samples, err := m.DataSamples()
	if err != nil {
		s.err = fmt.Errorf("%s: sample problem %s", s.StreamId, err)
		return
	}

	if samples == nil || len(samples) == 0 {
		s.err = fmt.Errorf("%s: nil or zero samples", s.StreamId)
		return
	}

	// should we reset the current filters
	if stream.HaveGap(m.Starttime()) {
		stream.Reset()
		stream.Condition(samples)
	}

	// assume packets are short (~10s on average), use the middle
	// of the packet as the time.
	// Time of max values is not exact to the sample.
	s.t = m.Starttime().Add(m.Endtime().Sub(m.Starttime()) / 2)

	s.pga, s.pgv = stream.Peaks(samples)

	// update the end time of the stream filter
	stream.Last = m.Endtime()

	return
}

// expire deletes any sites that have not been sending data.
func expire() {
	// age out DB values at this interval
	ticker := time.NewTicker(time.Minute).C
	var err error

	for {
		select {
		case <-ticker:
			if err = db.DeletePGA(time.Now().UTC().Add(notSending)); err != nil {
				log.Printf("WARN deleting old pga values from db: %s", err.Error())
			}

			if err = db.DeletePGV(time.Now().UTC().Add(notSending)); err != nil {
				log.Printf("WARN deleting old pgv values from db: %s", err.Error())
			}
		}
	}
}
