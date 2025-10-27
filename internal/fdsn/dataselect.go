// package fdsn is for Federation of Digital Seismic Networks web services.
package fdsn

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/schema"
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

// nslcReg: FDSN spec allows all ascii, but we'll only allow alpha, number, _,-, ?, *, "," and "--" (exactly 2 hyphens only)
var nslcReg = regexp.MustCompile(`^([\w*?,]+(?:-[\w*?,]+)*|--)$`)          // space not allowed
var eventTypeReg = regexp.MustCompile(`^([\w*?, ]+(?:[ -][\w*?,]+)*|--)$`) // space allowed

// nslcRegPassPattern: This is beyond FDSN spec.
// Any NSLC regex string doesn't match this pattern we knew it won't generate any results.
var nslcRegPassPattern = regexp.MustCompile(`^(\^[A-Z0-9\*\?\.]{2,6}\$)(\|?(\^[A-Z0-9\*\?\.]{2,6}\$))*$`) // "^WEL$|^VIZ$"

type DataSelect struct {
	StartTime   WsDateTime `schema:"starttime"` // limit to data on or after the specified start time.
	EndTime     WsDateTime `schema:"endtime"`   // limit to data on or before the specified end time.
	Network     []string   `schema:"network"`   // network name of data to query
	Station     []string   `schema:"station"`   // station name of data to query
	Location    []string   `schema:"location"`  // location name of data to query
	Channel     []string   `schema:"channel"`   // channel number of data to query
	Format      string     `schema:"format"`
	LongestOnly bool       `schema:"longestonly"`
	NoData      int        `schema:"nodata"` // Select status code for “no data”, either ‘204’ (default) or ‘404’.
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
						return errors.New("nodata must be 204 or 404")
					}
				}
			}
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 6 {
			return fmt.Errorf("incorrect number of fields in dataselect query POST body, expected 6 but observed: %d", len(fields))
		}

		startTime := WsDateTime{}
		if err := startTime.UnmarshalText([]byte(fields[4])); err != nil {
			return err
		}

		endTime := WsDateTime{}
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
			return DataSelect{}, fmt.Errorf("invalid %s value", key)
		}
	}

	err := decoder.Decode(&e, v)
	if err != nil {
		return DataSelect{}, err
	}

	if e.Format != "miniseed" {
		return DataSelect{}, fmt.Errorf("only \"miniseed\" format is supported")
	}

	if e.LongestOnly {
		return DataSelect{}, fmt.Errorf("query for longest only is not supported")
	}

	if e.NoData != 204 && e.NoData != 404 {
		return DataSelect{}, errors.New("nodata must be 204 or 404")
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
		return DataSelect{}, errors.New("startime parameter must be present")
	}
	if e.EndTime.IsZero() {
		return DataSelect{}, errors.New("endtime parameter must be present")
	}
	return e, nil
}

// regexp returns DataSearch with regexp strings that represents the search parameters.  It converts
// the '*', '?', ' ' and '--' characters to their regular expression equivalents for pattern matching with Postgres POSIX regexp.
func (d *DataSelect) Regexp() (DataSearch, error) {
	ne, err := toPattern(d.Network, false)
	if err != nil {
		return DataSearch{}, fmt.Errorf("invalid network parameter: %s", err.Error())
	}

	st, err := toPattern(d.Station, false)
	if err != nil {
		return DataSearch{}, fmt.Errorf("invalid station parameter: %s", err.Error())
	}

	ch, err := toPattern(d.Channel, false)
	if err != nil {
		return DataSearch{}, fmt.Errorf("invalid channel parameter: %s", err.Error())
	}

	lo, err := toPattern(d.Location, true)
	if err != nil {
		return DataSearch{}, fmt.Errorf("invalid location parameter: %s", err.Error())
	}

	return DataSearch{
		Start:    d.StartTime.Time,
		End:      d.EndTime.Time,
		Network:  ne,
		Station:  st,
		Location: lo,
		Channel:  ch,
	}, nil
}

func toPattern(params []string, emptyDash bool) (string, error) {
	newParams, err := GenRegex(params, emptyDash, false)
	if err != nil {
		return "", err
	}
	return strings.Join(newParams, `|`), nil
}

func GenRegex(input []string, emptyDash bool, allowSpace bool) ([]string, error) {
	if len(input) == 0 {
		return nil, nil
	}

	// FDSN spec: all ASCII chars are allowed, and only ? and * has special meaning.
	result := make([]string, 0)
	for _, s := range input {
		if s == "" {
			continue
		}

		var matched bool

		if allowSpace {
			matched = eventTypeReg.MatchString(s)
		} else {
			matched = nslcReg.MatchString(s)
		}

		if !matched {
			return nil, fmt.Errorf("invalid parameter:'%s'", s)
		}

		var r string

		if emptyDash && s == "--" {
			// "--" represents blank location which should be saved as 2 white spaces.
			r = `^\s{2}$`
		} else {
			s = strings.Replace(s, "*", ".*", -1)
			s = strings.Replace(s, "?", ".", -1)
			r = "^" + s + "$"
		}

		result = append(result, r)
	}

	return result, nil
}

func WillBeEmpty(s string) bool {
	// A query pattern could contains multiple patterns joined by "|", we check one by one
	for _, t := range strings.Split(s, "|") {
		// If a query doesn't match any of the patterns below,
		//   the query will be empty result because it contains unwanted characters.
		if !(t == `^\s{2}$` || t == "--" || nslcRegPassPattern.MatchString(t)) {
			return true
		}
	}
	return false
}
