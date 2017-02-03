// +build integration

package sc3ml_test

import (
	. "github.com/GeoNet/fdsn/internal/kit/sc3ml"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

// TestUnmarshalIntegration will list and unmarshal all .xml files
// under /work/seismcompml07-test and make sure basic quake facts
// are not zero values.  Run using:
//
//     go test -tags=integration
func TestUnmarshalIntegration(t *testing.T) {
	var s Seiscomp
	var err error
	var f *os.File
	var b []byte
	var files []os.FileInfo

	if files, err = ioutil.ReadDir("/work/seismcompml07-test"); err != nil {
		t.Fatal(err)
	}

	for _, fi := range files {
		if strings.HasSuffix(fi.Name(), ".xml") {

			if f, err = os.Open("/work/seismcompml07-test/" + fi.Name()); err != nil {
				t.Fatal(err)
			}
			defer f.Close()

			if b, err = ioutil.ReadAll(f); err != nil {
				t.Fatal(err)
			}
			f.Close()

			if err = Unmarshal(b, &s); err != nil {
				t.Fatal(err)
			}

			if len(s.EventParameters.Events) != 1 {
				t.Errorf("should have found 1 event for %s.", fi.Name())
			}

			e := s.EventParameters.Events[0]

			if e.PublicID == "" {
				t.Errorf("%s empty publicID", fi.Name())
			}

			if e.PreferredOrigin.Latitude.Value == 0.0 {
				t.Errorf("%s zero for Latitude", fi.Name())
			}

			if e.PreferredOrigin.Longitude.Value == 0.0 {
				t.Errorf("%s zero for Longitude", fi.Name())
			}

			// Depth is occasionally zero
			// if e.PreferredOrigin.Depth.Value == 0.0 {
			// 	t.Errorf("%s zero for Depth", fi.Name())
			// }

			if e.PreferredMagnitude.Magnitude.Value == 0.0 {
				t.Errorf("%s zero for Magnitude", fi.Name())
			}

			if e.PreferredOrigin.Time.Value.IsZero() {
				t.Errorf("%s zero for origin time", fi.Name())
			}

		}
	}

}
