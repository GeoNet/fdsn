package main

import (
	"math"
)

/*

Real-time filtering of incoming strong motion feeds via a set of first order filters to de-mean and convert the signal to velocity.
The actual filters used depend on the input signal gain (in counts/m/sec^2) and a high-pass filter parameter _q_.

Continuous Monitoring of Ground-Motion Parameters by Hiroo Kanamori, Philip Maechling, and Egill Hauksson

http://authors.library.caltech.edu/37034/1/311.full.pdf

*/

type HighPass struct {
	a, b float64 // filter coeffs
	x, y float64 // previous input & output
}

func NewHighPass(gain float64, q float64) *HighPass {
	return &HighPass{
		a: func() float64 {
			if gain != 0.0 {
				return (1.0 + q) / (2.0 * gain)
			}
			return 0.0
		}(),
		b: q,
		x: 0.0,
		y: math.NaN(),
	}
}

func (f *HighPass) Reset() {
	f.y = math.NaN()
}

func (f *HighPass) Set(y float64) {
	f.y = y
}

func (f *HighPass) Sample(x float64) float64 {
	var y float64

	if math.IsNaN(f.y) {
		f.x = x
		f.y = 0.0
	}

	y = (f.a*(x-f.x) + f.b*f.y)

	f.x, f.y = x, y

	return y
}

type Integrator struct {
	a, b float64 // filter coeffs
	x, y float64 // previous input & output
}

func NewIntegrator(gain float64, dt float64, q float64) *Integrator {
	return &Integrator{
		a: func() float64 {
			if gain != 0.0 {
				return (1.0 + q) * dt / (4.0 * gain)
			}
			return 0.0
		}(),
		b: q,
		x: 0.0,
		y: math.NaN(),
	}
}

func (f *Integrator) Reset() {
	f.y = math.NaN()
}

func (f *Integrator) Set(y float64) {
	f.y = y
}

func (f *Integrator) Sample(x float64) float64 {
	var y float64

	if math.IsNaN(f.y) {
		f.x = x
		f.y = 0.0
	}

	y = f.a*(x+f.x) + f.b*f.y

	f.x, f.y = x, y

	return y
}
