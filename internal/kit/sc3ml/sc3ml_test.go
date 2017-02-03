package sc3ml_test

import (
	. "github.com/GeoNet/fdsn/internal/kit/sc3ml"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

/*
Versions of the input files are created by editing the source file and changing
the version.  The validating them using the XSDs:

    xmllint --noout --schema sc3ml_0.7.xsd 2015p768477_0.7.xml
    xmllint --noout --schema sc3ml_0.8.xsd 2015p768477_0.8.xml
*/
func TestUnmarshal(t *testing.T) {
	for _, input := range []string{"etc/2015p768477_0.7.xml", "etc/2015p768477_0.8.xml"} {
		var s Seiscomp
		var err error
		var f *os.File
		var b []byte

		if f, err = os.Open(input); err != nil {
			t.Fatal(err)
		}

		if b, err = ioutil.ReadAll(f); err != nil {
			f.Close()
			t.Fatal(err)
		}
		f.Close()

		if err = Unmarshal(b, &s); err != nil {
			t.Errorf("%s: %s", input, err.Error())
		}

		if len(s.EventParameters.Events) != 1 {
			t.Errorf("%s: should have found 1 event.", input)
		}

		e := s.EventParameters.Events[0]

		if e.PublicID != "2015p768477" {
			t.Errorf("%s: expected publicID 2015p768477 got %s", input, e.PublicID)
		}

		if e.Type != "earthquake" {
			t.Errorf("%s: expected type earthquake got %s", input, e.Type)
		}

		if e.PreferredOriginID != "NLL.20151012224503.620592.155845" {
			t.Errorf("%s: expected preferredOriginID NLL.20151012224503.620592.155845 got %s", input, e.PreferredOriginID)
		}

		if e.PreferredOrigin.PublicID != "NLL.20151012224503.620592.155845" {
			t.Errorf("%s: expected NLL.20151012224503.620592.155845 got %s", input, e.PreferredOrigin.PublicID)
		}

		if e.PreferredOrigin.Time.Value.Format(time.RFC3339Nano) != "2015-10-12T08:05:01.717692Z" {
			t.Errorf("%s: expected 2015-10-12T08:05:01.717692Z, got %s", input, e.PreferredOrigin.Time.Value.Format(time.RFC3339Nano))
		}

		if e.PreferredOrigin.Latitude.Value != -40.57806609 {
			t.Errorf("%s: Latitude expected -40.57806609 got %f", input, e.PreferredOrigin.Latitude.Value)
		}
		if e.PreferredOrigin.Latitude.Uncertainty != 1.922480006 {
			t.Errorf("%s: Latitude uncertainty expected 1.922480006 got %f", input, e.PreferredOrigin.Latitude.Uncertainty)
		}

		if e.PreferredOrigin.Longitude.Value != 176.3257242 {
			t.Errorf("%s: Longitude expected 176.3257242 got %f", input, e.PreferredOrigin.Longitude.Value)
		}
		if e.PreferredOrigin.Longitude.Uncertainty != 3.435738791 {
			t.Errorf("%s: Longitude uncertainty expected 3.435738791 got %f", input, e.PreferredOrigin.Longitude.Uncertainty)
		}

		if e.PreferredOrigin.Depth.Value != 23.28125 {
			t.Errorf("%s: Depth expected 23.28125 got %f", input, e.PreferredOrigin.Depth.Value)
		}
		if e.PreferredOrigin.Depth.Uncertainty != 3.575079654 {
			t.Errorf("%s: Depth uncertainty expected 3.575079654 got %f", input, e.PreferredOrigin.Depth.Uncertainty)
		}

		if e.PreferredOrigin.MethodID != "NonLinLoc" {
			t.Errorf("%s: MethodID expected NonLinLoc got %s", input, e.PreferredOrigin.MethodID)
		}

		if e.PreferredOrigin.EarthModelID != "nz3drx" {
			t.Errorf("%s: EarthModelID expected NonLinLoc got %s", input, e.PreferredOrigin.EarthModelID)
		}

		if e.PreferredOrigin.Quality.StandardError != 0.5592857863 {
			t.Errorf("%s: StandardError expected 0.5592857863 got %f", input, e.PreferredOrigin.Quality.StandardError)
		}

		if e.PreferredOrigin.Quality.AzimuthalGap != 166.4674465 {
			t.Errorf("%s: AzimuthalGap expected 166.4674465 got %f", input, e.PreferredOrigin.Quality.AzimuthalGap)
		}

		if e.PreferredOrigin.Quality.MinimumDistance != 0.1217162272 {
			t.Errorf("%s: MinimumDistance expected 0.1217162272 got %f", input, e.PreferredOrigin.Quality.MinimumDistance)
		}

		if e.PreferredOrigin.Quality.UsedPhaseCount != 44 {
			t.Errorf("%s: UsedPhaseCount expected 44 got %d", input, e.PreferredOrigin.Quality.UsedPhaseCount)
		}

		if e.PreferredOrigin.Quality.UsedStationCount != 32 {
			t.Errorf("%s: UsedStationCount expected 32 got %d", input, e.PreferredOrigin.Quality.UsedStationCount)
		}

		var found bool
		for _, v := range e.PreferredOrigin.Arrivals {
			if v.PickID == "Pick#20151012081200.115203.26387" {
				found = true
				if v.Phase != "P" {
					t.Errorf("%s: expected P got %s", input, v.Phase)
				}

				if v.Azimuth != 211.917806 {
					t.Errorf("%s: azimuth expected 211.917806 got %f", input, v.Azimuth)
				}

				if v.Distance != 0.1217162272 {
					t.Errorf("%s: distance expected 0.1217162272 got %f", input, v.Distance)
				}

				if v.Weight != 1.406866218 {
					t.Errorf("%s: weight expected 1.406866218 got %f", input, v.Weight)
				}

				if v.TimeResidual != -0.01664948232 {
					t.Errorf("%s: time residual expected -0.01664948232 got %f", input, v.TimeResidual)
				}

				if v.Pick.WaveformID.NetworkCode != "NZ" {
					t.Errorf("%s: Pick.WaveformID.NetworkCode expected NZ, got %s", input, v.Pick.WaveformID.NetworkCode)
				}

				if v.Pick.WaveformID.StationCode != "BFZ" {
					t.Errorf("%s: Pick.WaveformID.StationCode expected BFZ, got %s", input, v.Pick.WaveformID.StationCode)
				}

				if v.Pick.WaveformID.LocationCode != "10" {
					t.Errorf("%s: Pick.WaveformID.LocationCode expected 10, got %s", input, v.Pick.WaveformID.LocationCode)
				}

				if v.Pick.WaveformID.ChannelCode != "HHN" {
					t.Errorf("%s: Pick.WaveformID.ChannelCode expected HHN, got %s", input, v.Pick.WaveformID.ChannelCode)
				}

				if v.Pick.EvaluationMode != "manual" {
					t.Errorf("%s: Pick.WaveformID.EvaluationMode expected manual got %s", input, v.Pick.EvaluationMode)
				}

				if v.Pick.EvaluationStatus != "" {
					t.Errorf("%s: Pick.WaveformID.EvaluationStatus expected empty string got %s", input, v.Pick.EvaluationStatus)
				}

				if v.Pick.Time.Value.Format(time.RFC3339Nano) != "2015-10-12T08:05:06.792207Z" {
					t.Errorf("%s: Pick.Time expected 2015-10-12T08:05:06.792207Z got %s", input, v.Pick.Time.Value.Format(time.RFC3339Nano))
				}
			}

		}
		if !found {
			t.Error("didn't find PickID Pick#20151012081200.115203.26387")
		}

		if e.PreferredMagnitude.Type != "M" {
			t.Errorf("%s: e.PreferredMagnitude.Type expected M got %s", input, e.PreferredMagnitude.Type)
		}
		if e.PreferredMagnitude.Magnitude.Value != 5.691131913 {
			t.Errorf("%s: magnitude expected 5.691131913 got %f", input, e.PreferredMagnitude.Magnitude.Value)
		}
		if e.PreferredMagnitude.Magnitude.Uncertainty != 0 {
			t.Errorf("%s: uncertainty expected 0 got %f", input, e.PreferredMagnitude.Magnitude.Uncertainty)
		}
		if e.PreferredMagnitude.StationCount != 171 {
			t.Errorf("%s: e.PreferredMagnitude.StationCount expected 171 got %d", input, e.PreferredMagnitude.StationCount)
		}
		if e.PreferredMagnitude.MethodID != "weighted average" {
			t.Errorf("%s: MethodID expected weighted average got %s", input, e.PreferredMagnitude.MethodID)
		}

		found = false

		for _, m := range e.PreferredOrigin.Magnitudes {
			if m.PublicID == "Magnitude#20151012224509.743338.156745" {
				found = true

				if m.Type != "ML" {
					t.Error("m.Type expected ML, got ", input, m.Type)
				}
				if m.Magnitude.Value != 6.057227661 {
					t.Errorf("%s: magnitude expected 6.057227661 got %f", input, m.Magnitude.Value)
				}
				if m.Magnitude.Uncertainty != 0.2576927171 {
					t.Errorf("%s: Uncertainty expected 0.2576927171 got %f", input, m.Magnitude.Uncertainty)
				}
				if m.StationCount != 23 {
					t.Errorf("%s: m.StationCount expected 23 got %d", input, m.StationCount)
				}
				if m.MethodID != "trimmed mean" {
					t.Errorf("%s: m.MethodID expected trimmed mean got %s", input, m.MethodID)
				}

				if !(len(m.StationMagnitudeContributions) > 1) {
					t.Error("expected more than 1 StationMagnitudeContribution")
				}

				var foundSM bool

				for _, s := range m.StationMagnitudeContributions {
					if s.StationMagnitudeID == "StationMagnitude#20151012224509.743511.156746" {
						foundSM = true

						if s.Weight != 1.0 {
							t.Errorf("%s: Weight expected 1.0 got %f", input, s.Weight)
						}

						if s.StationMagnitude.Magnitude.Value != 6.096018735 {
							t.Errorf("%s: StationMagnitude.Magnitude.Value expected 6.096018735 got %f", input, s.StationMagnitude.Magnitude.Value)
						}

						if s.StationMagnitude.Type != "ML" {
							t.Errorf("%s: StationMagnitude.Type expected ML got %s", input, s.StationMagnitude.Type)
						}

						if s.StationMagnitude.WaveformID.NetworkCode != "NZ" {
							t.Errorf("%s: Pick.WaveformID.NetworkCode expected NZ, got %s", input, s.StationMagnitude.WaveformID.NetworkCode)
						}

						if s.StationMagnitude.WaveformID.StationCode != "ANWZ" {
							t.Errorf("%s: Pick.WaveformID.StationCode expected ANWZ, got %s", input, s.StationMagnitude.WaveformID.StationCode)
						}

						if s.StationMagnitude.WaveformID.LocationCode != "10" {
							t.Errorf("%s: Pick.WaveformID.LocationCode expected 10, got %s", input, s.StationMagnitude.WaveformID.LocationCode)
						}

						if s.StationMagnitude.WaveformID.ChannelCode != "EH" {
							t.Errorf("%s: Pick.WaveformID.ChannelCode expected EH, got %s", input, s.StationMagnitude.WaveformID.ChannelCode)
						}

						if s.StationMagnitude.Amplitude.Amplitude.Value != 21899.94892 {
							t.Errorf("%s: Amplitude.Value expected 21899.94892 got %f", input, s.StationMagnitude.Amplitude.Amplitude.Value)
						}
					}
				}
				if !foundSM {
					t.Error("did not find StationMagnitudeContrib StationMagnitude#20151012224509.743511.156746")
				}
			}
		}

		if !found {
			t.Error("did not find magnitude smi:scs/0.7/Origin#20131202033820.196288.25287#netMag.MLv")
		}

		if e.ModificationTime.Format(time.RFC3339Nano) != "2015-10-12T22:46:41.228824Z" {
			t.Errorf("%s: Modification time expected 2015-10-12T22:46:41.228824Z got %s", input, e.ModificationTime.Format(time.RFC3339Nano))
		}
	}
}

func TestDecodeSC3ML07CMT(t *testing.T) {
	for _, input := range []string{"etc/2016p408314-201606010431276083_0.7.xml", "etc/2016p408314-201606010431276083_0.8.xml"} {
		var s Seiscomp
		var err error
		var f *os.File
		var b []byte

		if f, err = os.Open(input); err != nil {
			t.Fatal(err)
		}

		if b, err = ioutil.ReadAll(f); err != nil {
			f.Close()
			t.Fatal(err)
		}
		f.Close()

		if err = Unmarshal(b, &s); err != nil {
			t.Errorf("%s: %s", input, err.Error())
		}

		if len(s.EventParameters.Events) != 1 {
			t.Errorf("%s: should have found 1 event.", input)
		}

		e := s.EventParameters.Events[0]

		if e.PublicID != "2016p408314" {
			t.Errorf("%s: expected publicID 2016p408314 got %s", input, e.PublicID)
		}

		if e.Type != "earthquake" {
			t.Errorf("%s: expected type earthquake got %s", input, e.Type)
		}

		if e.PreferredOrigin.Time.Value.Format(time.RFC3339Nano) != "2016-05-31T01:50:12.062388Z" {
			t.Errorf("%s: expected 2016-05-31T01:50:12.062388Z, got %s", input, e.PreferredOrigin.Time.Value.Format(time.RFC3339Nano))
		}

		if e.PreferredOrigin.Latitude.Value != -45.19537735 {
			t.Errorf("%s: Latitude expected -45.19537735 got %f", input, e.PreferredOrigin.Latitude.Value)
		}

		if e.PreferredOrigin.Longitude.Value != 167.3780823 {
			t.Errorf("%s: Longitude expected 167.3780823 got %f", input, e.PreferredOrigin.Longitude.Value)
		}

		if e.PreferredOrigin.Depth.Value != 100.126976 {
			t.Errorf("%s: Depth expected 100.126976 got %f", input, e.PreferredOrigin.Depth.Value)
		}

		if e.PreferredOrigin.MethodID != "LOCSAT" {
			t.Errorf("%s: MethodID expected LOCSAT got %s", input, e.PreferredOrigin.MethodID)
		}

		if e.PreferredOrigin.EarthModelID != "iasp91" {
			t.Errorf("%s: EarthModelID expected iasp91 got %s", input, e.PreferredOrigin.EarthModelID)
		}

		if e.PreferredOrigin.Quality.AzimuthalGap != 186.5389404 {
			t.Errorf("%s: AzimuthalGap expected 186.5389404 got %f", input, e.PreferredOrigin.Quality.AzimuthalGap)
		}

		if e.PreferredOrigin.Quality.MinimumDistance != 0.3124738038 {
			t.Errorf("%s: MinimumDistance expected 0.3124738038 got %f", input, e.PreferredOrigin.Quality.MinimumDistance)
		}

		if e.PreferredOrigin.Quality.UsedPhaseCount != 18 {
			t.Errorf("%s: UsedPhaseCount expected 44 got %d", input, e.PreferredOrigin.Quality.UsedPhaseCount)
		}

		if e.PreferredOrigin.Quality.UsedStationCount != 14 {
			t.Errorf("%s: UsedStationCount expected 32 got %d", input, e.PreferredOrigin.Quality.UsedStationCount)
		}

		if e.PreferredMagnitude.Magnitude.Value != 4.452756951 {
			t.Errorf("%s: Magnitude expected 4.452756951 got %f", input, e.PreferredMagnitude.Magnitude.Value)
		}

		if e.PreferredMagnitude.Type != "Mw" {
			t.Errorf("%s: Magnitude type expected Mw got %s", input, e.PreferredMagnitude.Type)
		}

		if e.PreferredMagnitude.StationCount != 19 {
			t.Errorf("%s: Expected StationCount 19 gor %d", input, e.PreferredMagnitude.StationCount)
		}
	}
}

func BenchmarkUnmarshalSeiscompml(b *testing.B) {
	var s Seiscomp
	var err error
	var f *os.File
	var by []byte

	if f, err = os.Open("etc/2015p768477_0.7.xml"); err != nil {
		b.Fatal(err)
	}
	defer f.Close()

	if by, err = ioutil.ReadAll(f); err != nil {
		b.Fatal(err)
	}

	if err = Unmarshal(by, &s); err != nil {
		b.Fatal(err)
	}

	for n := 0; n < b.N; n++ {
		// ignore errors
		_ = Unmarshal(by, &s)
	}
}
