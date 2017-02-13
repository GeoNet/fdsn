package main

import (
	"github.com/GeoNet/delta"
	"math"
	"time"
)

const GRAVITY = 9.80665

// running acceleration stream state information
type Stream struct {
	delta.ShakeNZStream

	HighPass   *HighPass
	Integrator *Integrator
	Last       time.Time
}

// streamEqual returns true if the config params of a and b are equal.
func streamEqual(a, b *Stream) bool {
	if a == nil {
		return false
	}
	if b == nil {
		return false
	}
	if a.StreamId != b.StreamId {
		return false
	}
	if a.Latitude != b.Latitude {
		return false
	}
	if a.Longitude != b.Longitude {
		return false
	}
	if a.SampleRate != b.SampleRate {
		return false
	}
	if a.Gain != b.Gain {
		return false
	}
	if a.Q != b.Q {
		return false
	}

	return true
}

// reset the stream
func (s *Stream) Reset() {
	if s.HighPass != nil {
		s.HighPass.Reset()
	}
	if s.Integrator != nil {
		s.Integrator.Reset()
	}
}

// detect a break in the stream - signals a reset required
func (s *Stream) HaveGap(at time.Time) bool {
	return math.Abs(at.Sub(s.Last).Seconds()-1.0/s.SampleRate) > (0.5 / s.SampleRate)
}

// try and make the filter a little less spiky after a reset
func (s *Stream) Condition(samples []int32) {
	for i := range samples {
		s.Sample(samples[len(samples)-i-1])
	}
}

// add a sample to the filter ...
func (s *Stream) Sample(sample int32) (float64, float64) {
	// acceleration
	a := s.HighPass.Sample(float64(sample))
	// velocity
	v := s.Integrator.Sample(a)
	// update units
	return 100.0 * a / GRAVITY, v * 100.0
}

// estimate the peak ground motions ...
func (s *Stream) Peaks(samples []int32) (float64, float64) {
	var pga, pgv float64
	for i := range samples {
		a, v := s.Sample(samples[i])
		if math.Abs(a) > pga {
			pga = math.Abs(a)
		}
		if math.Abs(v) > pgv {
			pgv = math.Abs(v)
		}
	}
	return pga, pgv
}
