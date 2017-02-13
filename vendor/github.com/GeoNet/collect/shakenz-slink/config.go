package main

import (
	"fmt"
	"github.com/GeoNet/delta"
	"github.com/golang/protobuf/proto"
	"io/ioutil"
	"log"
	"sync"
)

type config struct {
	sync.RWMutex
	m map[string]*Stream
}

// pull in public stream information from a protobuf config file
// TODO - load from a web service
func (c *config) load() error {
	log.Print("loading config")

	b, err := ioutil.ReadFile("shakenz-slink.pb")
	if err != nil {
		return err
	}

	var s delta.ShakeNZStreams
	err = proto.Unmarshal(b, &s)
	if err != nil {
		return err
	}

	var t = make(map[string]*Stream)

	// configure the filters ...
	for k, v := range s.Streams {
		n := Stream{ShakeNZStream: *v}

		// this will produce a de-meaned acceleration
		n.HighPass = NewHighPass(v.Gain, v.Q)

		// this will convert the stream to velocity
		n.Integrator = NewIntegrator(1.0, 1.0/v.SampleRate, v.Q)

		t[k] = &n
	}

	if t != nil && len(t) > 0 {
		c.Lock()
		// if the stream config hasn't changed then copy over the
		// existing one.  This avoids interrupting the filter with unnecessary gaps.
		for k, v := range t {
			if streamEqual(v, c.m[k]) {
				t[k] = c.m[k]
			}
		}
		c.m = t
		c.Unlock()

		log.Print("succesfully loaded config")

		// update the shaking sources in the DB.
		go c.save()

		return nil
	}

	return fmt.Errorf("trouble loading config found nil or len %d", 0)
}

// save source information in the DB.
func (c *config) save() {
	var d []delta.ShakeNZStream

	c.RLock()
	for _, v := range c.m {
		d = append(d, v.ShakeNZStream)
	}
	c.RUnlock()

	var err error

	for _, v := range d {
		log.Printf("config %s %s %s Gain:%f Q:%f", v.Source, v.Datalogger, v.Sensor, v.Gain, v.Q)
		if err = db.SaveSource(v.Source, v.Longitude, v.Latitude); err != nil {
			log.Printf("saving config for %s: %s", v.Source, err.Error())
		}
	}
}
