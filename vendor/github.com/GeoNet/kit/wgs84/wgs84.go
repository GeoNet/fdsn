// Package wgs84 is for distance, bearing, and locality calculations on the WGS84 ellipsoid.
package wgs84

import (
	"errors"
	"math"
)

const (
	pi           = math.Pi
	twopi        = math.Pi * 2.0
	maxLoopCount = 20
	eps          = 1.0e-23
)

// DistanceBearing calculates the distance in km and the bearing in degrees from lat1 lon1
// to lat2 lon2 on the WGS84 ellipsoid.
//
// Based on https://github.com/StefanSchroeder/Golang-Ellipsoid
func DistanceBearing(lat1, lon1, lat2, lon2 float64) (distance, bearing float64, err error) {
	lat1 = deg2rad(lat1)
	lon1 = deg2rad(lon1)
	lat2 = deg2rad(lat2)
	lon2 = deg2rad(lon2)

	a := 6378137.0         // equatorial WGS84
	f := 1 / 298.257223563 // inv flattening WGS84

	if lon1 < 0 {
		lon1 += twopi
	}

	if lon2 < 0 {
		lon2 += twopi
	}

	r := 1.0 - f

	clat1 := math.Cos(lat1)
	if clat1 == 0 {
		return 0.0, 0.0, errors.New("division by zero")
	}

	clat2 := math.Cos(lat2)
	if clat2 == 0 {
		return 0.0, 0.0, errors.New("division by zero")
	}

	tu1 := r * math.Sin(lat1) / clat1
	tu2 := r * math.Sin(lat2) / clat2
	cu1 := 1.0 / (math.Sqrt((tu1 * tu1) + 1.0))
	su1 := cu1 * tu1
	cu2 := 1.0 / (math.Sqrt((tu2 * tu2) + 1.0))
	s := cu1 * cu2
	baz := s * tu2
	faz := baz * tu1
	dlon := lon2 - lon1

	x := dlon
	cnt := 0

	var c2a, c, cx, cy, cz, d, del, e, sx, sy, y float64

	for true {
		sx = math.Sin(x)
		cx = math.Cos(x)
		tu1 = cu2 * sx
		tu2 = baz - (su1 * cu2 * cx)

		sy = math.Sqrt(tu1*tu1 + tu2*tu2)
		cy = s*cx + faz
		y = math.Atan2(sy, cy)
		var sa float64
		if sy == 0.0 {
			sa = 1.0
		} else {
			sa = (s * sx) / sy
		}

		c2a = 1.0 - (sa * sa)
		cz = faz + faz
		if c2a > 0.0 {
			cz = ((-cz) / c2a) + cy
		}
		e = (2.0 * cz * cz) - 1.0
		c = (((((-3.0 * c2a) + 4.0) * f) + 4.0) * c2a * f) / 16.0
		d = x
		x = ((e*cy*c+cz)*sy*c + y) * sa
		x = (1.0-c)*x*f + dlon
		del = d - x

		if math.Abs(del) <= eps {
			break
		}
		cnt++
		if cnt > maxLoopCount {
			break
		}

	}

	faz = math.Atan2(tu1, tu2)
	baz = math.Atan2(cu1*sx, (baz*cx-su1*cu2)) + pi
	x = math.Sqrt(((1.0/(r*r))-1.0)*c2a+1.0) + 1.0
	x = (x - 2.0) / x
	c = 1.0 - x
	c = ((x*x)/4.0 + 1.0) / c
	d = ((0.375 * x * x) - 1.0) * x
	x = e * cy

	s = 1.0 - e - e
	s = ((((((((sy * sy * 4.0) - 3.0) * s * cz * d / 6.0) - x) * d / 4.0) + cz) * sy * d) + y) * c * a * r

	// adjust azimuth to (-180,180)
	if faz < 0 {
		faz += twopi
	}
	if faz >= twopi {
		faz -= twopi
	}

	distance, bearing = s/1000.0, faz
	bearing = rad2deg(bearing)

	return
}

func deg2rad(d float64) (r float64) {
	return d * pi / 180.0
}

func rad2deg(d float64) (r float64) {
	return d * 180.0 / pi
}
