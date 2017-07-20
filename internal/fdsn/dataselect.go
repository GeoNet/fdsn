// package fdsn is for Federation of Digital Seismic Networks web services.
package fdsn

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/gorilla/schema"
	"io"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var decoder = schema.NewDecoder()

// dataselect abbreviations
var abbreviations = map[string]string{
	"net":   "network",
	"sta":   "station",
	"loc":   "location",
	"cha":   "channel",
	"start": "starttime",
	"end":   "endtime",
}

var dataSelectNotSupported = map[string]bool{
	"quality":        true,
	"minuimumlength": true,
}

type DataSelect struct {
	StartTime   Time     `schema:"starttime"` // limit to data on or after the specified start time.
	EndTime     Time     `schema:"endtime"`   // limit to data on or before the specified end time.
	Network     []string `schema:"network"`   // network name of data to query
	Station     []string `schema:"station"`   // station name of data to query
	Location    []string `schema:"location"`  // location name of data to query
	Channel     []string `schema:"channel"`   // channel number of data to query
	Format      string   `schema:"format"`
	LongestOnly bool     `schema:"longestonly"`
	NoData      int      `schema:"nodata"` // Select status code for “no data”, either ‘204’ (default) or ‘404’.
}

type Time struct {
	time.Time
}

type DataSearch struct {
	Start, End                          time.Time
	Network, Station, Location, Channel string
}

func init() {
	// Handle comma separated parameters (eg: net, sta, loc, cha, etc)
	decoder.RegisterConverter([]string{}, func(input string) reflect.Value {
		return reflect.ValueOf(strings.Split(input, ","))
	})
}

/*
parses the time in text as per the FDSN spec.  Pads text for parsing with
time.RFC3339Nano.  Accepted formats are (UTC):
   YYYY-MM-DDTHH:MM:SS.ssssss
   YYYY-MM-DDTHH:MM:SS
   YYYY-MM-DD

Implements the encoding.TextUnmarshaler interface.
*/
func (t *Time) UnmarshalText(text []byte) (err error) {
	s := string(text)
	l := len(s)
	if len(s) < 10 {
		return fmt.Errorf("invalid time format: %s", s)
	}

	if l >= 19 && l <= 26 && l != 20 { // length 20: "YYYY-MM-DDTHH:MM:SS." invalid
		s = s + ".000000000Z"[(l-19):] // "YYYY-MM-DDTHH:MM:SS" append to nano
	} else if l == 10 {
		s = s + "T00:00:00.000000000Z" // YYYY-MM-DD
	} else {
		return fmt.Errorf("invalid time format: %s", s)
	}
	t.Time, err = time.Parse(time.RFC3339Nano, s)
	return
}

// ParesDataSelectGet parses the FDSN dataselect parameters in r from a
// dataselect POST request.
func ParseDataSelectPost(r io.Reader, d *[]DataSelect) error {
	scanner := bufio.NewScanner(r)
	noData := 204

	for scanner.Scan() {

		line := scanner.Text()
		// ignore any blank lines or lines with "=", we don't use any of these parameters and "=" is otherwise invalid
		if len(line) == 0 {
			continue
		}

		if strings.Contains(line, "=") {
			if tokens := strings.Split(line, "="); len(tokens) == 2 {
				switch tokens[0] {
				case "nodata":
					var err error
					if noData, err = strconv.Atoi(strings.TrimSpace(tokens[1])); err != nil {
						return errors.New("error nodata value:" + err.Error())
					}

					if noData != 204 && noData != 404 {
						return errors.New("nodata must be 204 or 404.")
					}
				}
			}
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 6 {
			return fmt.Errorf("incorrect number of fields in dataselect query POST body, expected 6 but observed: %d", len(fields))
		}

		startTime := Time{}
		if err := startTime.UnmarshalText([]byte(fields[4])); err != nil {
			return err
		}

		endTime := Time{}
		if err := endTime.UnmarshalText([]byte(fields[5])); err != nil {
			return err
		}

		*d = append(*d,
			DataSelect{
				StartTime: startTime,
				EndTime:   endTime,
				Network:   []string{fields[0]},
				Station:   []string{fields[1]},
				Location:  []string{fields[2]},
				Channel:   []string{fields[3]},
				Format:    "miniseed",
				NoData:    noData,
			})
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

// ParesDataSelectGet parses the FDSN dataselect parameters in v from a
// dataselect GET request.
func ParseDataSelectGet(v url.Values) (DataSelect, error) {
	e := DataSelect{
		Format: "miniseed",
		NoData: 204,
	}

	// convert all abbreviated params to their expanded form
	for abbrev, expanded := range abbreviations {
		if val, ok := v[abbrev]; ok {
			v[expanded] = val
			delete(v, abbrev)
		}
	}

	// Valid parameter values.
	// Note: Since we're only checking the first occurrence of a parameter,
	//   so we're not handling "parameter submitted multiple times" - it might pass or fail.
	// (According to spec 1.1 Page 10 top section)

	for key, val := range v {
		if _, ok := dataSelectNotSupported[key]; ok {
			return DataSelect{}, fmt.Errorf("\"%s\" is not supported", key)
		}
		if len(val[0]) == 0 {
			return DataSelect{}, fmt.Errorf("Invalid %s value", key)
		}
	}

	err := decoder.Decode(&e, v)
	if err != nil {
		return DataSelect{}, err
	}

	if e.Format != "miniseed" {
		return DataSelect{}, fmt.Errorf("Only \"miniseed\" format is supported.")
	}

	if e.LongestOnly {
		return DataSelect{}, fmt.Errorf("Query for longest only is not supported.")
	}

	if e.NoData != 204 && e.NoData != 404 {
		return DataSelect{}, errors.New("nodata must be 204 or 404.")
	}

	// Defaults: as per spec we need to include any valid files in the search so use wildcards and broad time range
	if len(e.Network) == 0 {
		e.Network = []string{"*"}
	}
	if len(e.Station) == 0 {
		e.Station = []string{"*"}
	}
	if len(e.Location) == 0 {
		e.Location = []string{"*"}
	}
	if len(e.Channel) == 0 {
		e.Channel = []string{"*"}
	}

	if e.StartTime.IsZero() {
		e.StartTime.Time, err = time.Parse(time.RFC3339, "2000-01-01T00:00:00Z")
		if err != nil {
			return DataSelect{}, err
		}
	}

	if e.EndTime.IsZero() {
		e.EndTime.Time = time.Now().UTC()
	}
	return e, nil
}

// regexp returns DataSearch with regexp strings that represents the search parameters.  It converts
// the '*', '?', ' ' and '--' characters to their regular expression equivalents for pattern matching with Postgres POSIX regexp.
func (d *DataSelect) Regexp() DataSearch {
	return DataSearch{
		Start:    d.StartTime.Time,
		End:      d.EndTime.Time,
		Network:  toPattern(d.Network),
		Station:  toPattern(d.Station),
		Location: toPattern(d.Location),
		Channel:  toPattern(d.Channel),
	}
}

func toPattern(params []string) (out string) {
	var newParams []string
	for _, param := range params {
		newParam := strings.Replace(param, `*`, `\w*`, -1)
		newParam = strings.Replace(newParam, `?`, `\w{1}`, -1)
		// blank or missing locations, we convert spaces and two dashes to wildcards for the regexp
		newParam = strings.Replace(newParam, `--`, `\w{2}`, -1)
		newParam = strings.Replace(newParam, ` `, `\w{1}`, -1)
		newParams = append(newParams, `(^`+newParam+`$)`)
	}

	return strings.Join(newParams, `|`)
}
