package main

import (
	"math"
)

// algorithm to convert peak velocity in m/s into MMI intensity
type IntensityEquation interface {
	RawIntensity(vel float64) float64
}

// convert peak velocity in m/s into integer MMI intensity
func Intensity(ie IntensityEquation, vel float64) int32 {

	raw := ie.RawIntensity(vel)

	switch {
	case raw <= 1.0:
		return 1
	case raw >= 12.0:
		return 12
	default:
		return (int32)(math.Floor(raw + 0.5))
	}
}

// David J. Wald, Vincent Quitoriano, Thomas H. Heaton, and Hiroo Kanamori (1999),
// "Relationships between Peak Ground Acceleration, Peak Ground Velocity, and
// Modified Mercalli Intensity in California", Earthquake Spectra, Volume 15, No. 3, August 1999.
type WaldQuitorianoHeatonKanamori1999 struct{}

func (fn WaldQuitorianoHeatonKanamori1999) RawIntensity(vel float64) float64 {
	return 2.35 + 3.47*math.Log10(100.0*math.Abs(vel)+1.0e-9)
}

// L. Faenza and A. Michelini (2010),
// "Regression analysis of MCS Intensity and ground motion parameters in Italy and its application
// in ShakeMap", Geophysical Journal International, 180: 1138–1152.
type FaenzaMichelini2010 struct{}

func (fn FaenzaMichelini2010) RawIntensity(vel float64) float64 {
	return 5.11 + 2.35*math.Log10(100.0*math.Abs(vel)+1.0e-9)
}
