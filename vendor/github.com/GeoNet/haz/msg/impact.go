package msg

import (
	"encoding/json"
	"fmt"
	"regexp"
	"time"
)

var source = regexp.MustCompile(`^[a-zA-Z0-9\.\-]+$`)
var MeasuredAge = time.Duration(-60) * time.Minute // measured intensity messages older than this are not saved to the DB.
var future = time.Duration(10) * time.Second

// Intensity is for measured or reported intensity messages e.g.,
//  {
//     "Time": "2014-12-31T02:39:00Z",
//     "Longitude": 172,
//     "Latitude": -42.4,
//     "MMI": 4,
//     "Comment": "",
//     "Quality": "measured",
//     "Source": "test.test"
//  }
type Intensity struct {
	// Source is used to uniquely identify the intensity source.
	// 'measured' and 'reported' values are stored separately.
	// Use a prefix if to keep them distinct in major populations
	// and make sure sources are distinct wihin a population.
	// Good choices might be 'ios.xxx', 'android.xxx', 'NZ.xxx'.
	// Must match the regexp `^[a-zA-Z0-9\.\-]+$`
	Source    string
	Quality   string    // allowed values are 'measured' or 'reported'.
	Comment   string    // max length 140 char.
	MMI       int       // range 1 - 12
	Latitude  float64   //  WGS84, -90 to 90.
	Longitude float64   // WGS84, -180 to 180.
	Time      time.Time // date time ISO8601 UTC.
	err       error
}

func (i *Intensity) Err() error {
	return i.err
}

func (i *Intensity) SetErr(err error) {
	i.err = err
}

// Valid sets i.err if i is invalid.
// i.Comment is trimmed to 140 char.
func (i *Intensity) Valid() {
	if i.err != nil {
		return
	}

	if !source.MatchString(i.Source) {
		i.err = fmt.Errorf("invalid source: %s must match %s", i.Source, source.String())
	}

	if !(i.Quality == "measured" || i.Quality == "reported") {
		i.err = fmt.Errorf("invalid quality: %s", i.Quality)
	}

	if i.MMI < 1 || i.MMI > 12 {
		i.err = fmt.Errorf("invalid MMI: %d", i.MMI)
	}

	if len(i.Comment) > 139 {
		i.Comment = i.Comment[0:139]
	}

	return
}

// Old sets i.err if the intensity pointed to by i is older then 60 minutes.
func (i *Intensity) Old() {
	if i.err != nil {
		return
	}
	// No disctinction between measured and reported intensity at the moment.
	// We may need to allow slightly older reported intensity messages in the future?
	if i.Time.Before(time.Now().UTC().Add(MeasuredAge)) {
		i.err = fmt.Errorf("old message for %s", i.Source)
	}
	return
}

// Future sets i.err if the intensity pointed to be i is in the future.
func (i *Intensity) Future() {
	if i.err != nil {
		return
	}

	if i.Time.After(time.Now().UTC().Add(future)) {
		i.err = fmt.Errorf("future message for %s", i.Source)
	}

	return
}

func (i *Intensity) Decode(b []byte) {
	i.err = json.Unmarshal(b, i)
}

func (i *Intensity) Encode() ([]byte, error) {
	if i.err != nil {
		return nil, i.err
	}

	return json.Marshal(i)
}
