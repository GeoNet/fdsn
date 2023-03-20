package main

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/GeoNet/fdsn/internal/fdsn"
	"github.com/GeoNet/kit/aws/s3"
	"github.com/GeoNet/kit/weft"
	"github.com/GeoNet/kit/wgs84"
)

const (
	STATION_LEVEL_NETWORK   = 0
	STATION_LEVEL_STATION   = 1
	STATION_LEVEL_CHANNEL   = 2
	STATION_LEVEL_RESPONSE  = 3
	DEFAULT_RELOAD_INTERVAL = 300
	NZ_KM_DEGREE            = 111.0
	NO_DATA                 = 204

	BEFORE       = -1
	ONBEFOREEND  = 0
	ONAFTERSTART = 0
	AFTER        = 1

	REGEX_ANYTHING = "^.*$"
)

var stationAbbreviations = map[string]string{
	"net":    "network",
	"sta":    "station",
	"loc":    "location",
	"cha":    "channel",
	"start":  "starttime",
	"end":    "endtime",
	"minlat": "minlatitude",
	"maxlat": "maxlatitude",
	"minlon": "minlongitude",
	"maxlon": "maxlongitude",
	"lat":    "latitude",
	"lon":    "longitude",
}

// supported query parameters for the station service from http://www.fdsn.org/webservices/FDSN-WS-Specifications-1.1.pdf
type fdsnStationV1Parm struct {
	StartTime           Time     `schema:"starttime"`    // Limit to metadata epochs starting on or after the specified start time.
	EndTime             Time     `schema:"endtime"`      // Limit to metadata epochs ending on or before the specified end time.
	Network             []string `schema:"network"`      // Select one or more network codes. Can be SEED network codes or data center defined codes. Multiple codes are comma-separated.
	Station             []string `schema:"station"`      // Select one or more SEED station codes. Multiple codes are comma-separated.
	Location            []string `schema:"location"`     // Select one or more SEED location identifiers. Multiple identifiers are comma- separated. As a special case “--“ (two dashes) will be translated to a string of two space characters to match blank location IDs.
	Channel             []string `schema:"channel"`      // Select one or more SEED channel codes. Multiple codes are comma-separated.
	MinLatitude         float64  `schema:"minlatitude"`  // Limit to stations with a latitude larger than or equal to the specified minimum.
	MaxLatitude         float64  `schema:"maxlatitude"`  // Limit to stations with a latitude smaller than or equal to the specified maximum.
	MinLongitude        float64  `schema:"minlongitude"` // Limit to stations with a longitude larger than or equal to the specified minimum.
	MaxLongitude        float64  `schema:"maxlongitude"` // Limit to stations with a longitude smaller than or equal to the specified maximum.
	Level               string   `schema:"level"`        // Specify the level of detail for the results.
	Format              string   `schema:"format"`       // Format of result. Either "xml" or "text".
	IncludeAvailability bool     `schema:"includeavailability"`
	IncludeRestricted   bool     `schema:"includerestricted"`
	MatchTimeSeries     bool     `schema:"matchtimeseries"`
	Latitude            float64  `schema:"latitude"`
	Longitude           float64  `schema:"longitude"`
	MinRadius           float64  `schema:"minradius"`
	MaxRadius           float64  `schema:"maxradius"`
	StartBefore         Time     `schema:"startbefore"`
	StartAfter          Time     `schema:"startafter"`
	EndBefore           Time     `schema:"endbefore"`
	EndAfter            Time     `schema:"endafter"`
	NoData              int      `schema:"nodata"` // Select status code for “no data”, either ‘204’ (default) or ‘404’.
	startMode           int      // BEFORE, ONBEFOREEND, AFTER
	endMode             int      // BEFORE, ONAFTERSTART, AFTER
}

type fdsnStationV1Search struct {
	fdsnStationV1Parm
	LevelValue  int
	NetworkReg  []string
	StationReg  []string
	LocationReg []string
	ChannelReg  []string
}

type fdsnStationObj struct {
	fdsn     *FDSNStationXML
	modified time.Time
	sync.RWMutex
}

var (
	fdsnStationWadlFile []byte
	fdsnStationIndex    []byte
	fdsnStations        fdsnStationObj
	emptyDateTime       time.Time
	errNotModified      = fmt.Errorf("Not modified.")
	s3Bucket            string
	s3Meta              string
)

func init() {
	var err error
	var b bytes.Buffer

	t, err := template.New("t").ParseFiles("assets/tmpl/fdsn-ws-station.wadl")
	if err != nil {
		log.Printf("error parsing assets/tmpl/fdsn-ws-station.wadl: %s", err.Error())
	}
	err = t.ExecuteTemplate(&b, "body", os.Getenv("HOST_CNAME"))
	if err != nil {
		log.Printf("error executing assets/tmpl/fdsn-ws-station.wadl: %s", err.Error())
	}
	fdsnStationWadlFile = b.Bytes()

	fdsnStationIndex, err = ioutil.ReadFile("assets/fdsn-ws-station.html")
	if err != nil {
		log.Printf("error reading assets/fdsn-ws-station.html: %s", err.Error())
	}

	emptyDateTime = time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)

	s3Bucket = os.Getenv("STATION_XML_BUCKET")
	s3Meta = os.Getenv("STATION_XML_META_KEY")

	// Prepare the data source for station.
	// If there's no local file available then we'll have to download first.
	by := bytes.NewBuffer(nil)
	modified := zeroDateTime
	var s os.FileInfo
	if s, err = os.Stat("etc/" + s3Meta); err == nil {
		log.Println("Loading fdsn station xml file ", "etc/"+s3Meta)
		var f *os.File
		if f, err = os.Open("etc/" + s3Meta); err == nil {

			if _, err = io.Copy(by, f); err != nil {
				log.Println("Error copying station xml file", err)
			}
			if err = f.Close(); err != nil {
				log.Println("Error closing station xml file", err)
			}
			modified = s.ModTime()
		}
	}

	// Local file not exist, or got error while reading it.
	// Read from S3 instead.
	if err != nil {
		by, modified, err = downloadStationXML(zeroDateTime)
		if err != nil {
			// errNotModified wouldn't happen here
			log.Fatalf("Download from S3 error: %s\n", err.Error())
		}
	}

	fdsnStations, err = loadStationXML(by, modified)
	if err != nil {
		log.Fatalf("Error loading xml: %s\n", err)
	}
}

func parseStationV1Post(body string) ([]fdsnStationV1Search, error) {
	ret := []fdsnStationV1Search{}
	level := "station"
	format := "xml"

	scanner := bufio.NewScanner(strings.NewReader(body))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if tokens := strings.Split(line, "="); len(tokens) == 2 {
			switch tokens[0] {
			case "level":
				level = strings.TrimSpace(tokens[1])
				if _, err := levelValue(level); err != nil {
					return ret, err
				}
			case "format":
				format = strings.TrimSpace(tokens[1])
				if format != "xml" && format != "text" {
					return ret, errors.New("Invalid format")
				}
			}
		} else if tokens := strings.Fields(line); len(tokens) == 6 {
			// NET STA LOC CHA STARTTIME ENDTIME
			// IU COLA 00 LH? 2012-01-01T00:00:00 2012-01-01T12:00:00
			// IU ANMO 10 BH? 2013-07-01T00:00:00 2013-02-07T12:00:00
			v := url.Values{}
			v.Add("Network", tokens[0])
			v.Add("Station", tokens[1])
			v.Add("Location", tokens[2])
			v.Add("Channel", tokens[3])
			if strings.Compare(tokens[4], "*") != 0 {
				v.Add("StartTime", tokens[4])
			}
			if strings.Compare(tokens[5], "*") != 0 {
				v.Add("EndTime", tokens[5])
			}
			v.Add("Level", level)
			v.Add("format", format)

			p, err := parseStationV1(v)
			if err != nil {
				return ret, err
			}
			ret = append(ret, p)
		} else {
			return ret, fmt.Errorf("Invalid query format (POST).")
		}
	}

	if level == "response" && format == "text" {
		return []fdsnStationV1Search{}, fmt.Errorf("Text formats are only supported when level is net|sta|cha.")
	}

	return ret, nil
}

func parseStationV1(v url.Values) (fdsnStationV1Search, error) {
	// All query parameters are optional and float zero values overlap
	// with possible request ranges so the default is set to the max float val.
	p := fdsnStationV1Parm{
		MinLatitude:       math.MaxFloat64,
		MaxLatitude:       math.MaxFloat64,
		MinLongitude:      math.MaxFloat64,
		MaxLongitude:      math.MaxFloat64,
		Level:             "station",
		Format:            "xml",
		IncludeRestricted: true,
		StartTime:         Time{zeroDateTime},  // 0001-01-01T00:00:00
		EndTime:           Time{emptyDateTime}, // 9999-01-01T00:00:00
		StartBefore:       Time{emptyDateTime}, // 9999-01-01T00:00:00
		EndBefore:         Time{emptyDateTime}, // 9999-01-01T00:00:00
		StartAfter:        Time{emptyDateTime}, // 9999-01-01T00:00:00
		EndAfter:          Time{emptyDateTime}, // 9999-01-01T00:00:00
		Latitude:          math.MaxFloat64,
		Longitude:         math.MaxFloat64,
		MinRadius:         0.0,
		MaxRadius:         180.0,
		NoData:            NO_DATA,
	}

	for abbrev, expanded := range stationAbbreviations {
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
		if len(val[0]) == 0 {
			return fdsnStationV1Search{}, fmt.Errorf("Invalid %s value", key)
		}
	}

	err := decoder.Decode(&p, v)
	if err != nil {
		return fdsnStationV1Search{}, err
	}

	// Only xml and text is allowed.
	if p.Format != "xml" && p.Format != "text" {
		return fdsnStationV1Search{}, fmt.Errorf("Invalid format.")
	}

	if p.Level == "response" && p.Format == "text" {
		return fdsnStationV1Search{}, fmt.Errorf("Text formats are only supported when level is net|sta|cha.")
	}

	count := 0
	if p.StartTime.Time != zeroDateTime {
		count++
		p.startMode = ONBEFOREEND
	}
	if p.StartAfter.Time != emptyDateTime {
		count++
		p.startMode = AFTER
		p.StartTime = p.StartAfter
	}
	if p.StartBefore.Time != emptyDateTime {
		count++
		p.startMode = BEFORE
		p.StartTime = p.StartBefore
	}
	if count > 1 {
		return fdsnStationV1Search{}, fmt.Errorf("Only one of 'starttime', 'startafter', and 'startbefore' is allowed.")
	}

	count = 0
	if p.EndTime.Time != emptyDateTime {
		count++
		p.endMode = ONAFTERSTART
	}

	if p.EndAfter.Time != emptyDateTime {
		count++
		p.endMode = AFTER
		p.EndTime = p.EndAfter
	}

	if p.EndBefore.Time != emptyDateTime {
		count++
		p.endMode = BEFORE
		p.EndTime = p.EndBefore
	}

	if count > 1 {
		return fdsnStationV1Search{}, fmt.Errorf("Only one of 'endtime', 'endafter', and 'endbefore' is allowed.")
	}

	if p.IncludeAvailability {
		return fdsnStationV1Search{}, errors.New("include availability is not supported.")
	}

	if !p.IncludeRestricted {
		return fdsnStationV1Search{}, errors.New("exclude restricted is not supported.")
	}

	if p.MatchTimeSeries {
		return fdsnStationV1Search{}, errors.New("match time series is not supported.")
	}

	if p.NoData != 204 && p.NoData != 404 {
		return fdsnStationV1Search{}, errors.New("nodata must be 204 or 404.")
	}

	ne, err := fdsn.GenRegex(p.Network, false, false)
	if err != nil {
		return fdsnStationV1Search{}, err
	}
	st, err := fdsn.GenRegex(p.Station, false, false)
	if err != nil {
		return fdsnStationV1Search{}, err
	}
	ch, err := fdsn.GenRegex(p.Channel, false, false)
	if err != nil {
		return fdsnStationV1Search{}, err
	}
	lo, err := fdsn.GenRegex(p.Location, true, false)
	if err != nil {
		return fdsnStationV1Search{}, err
	}
	s := fdsnStationV1Search{
		fdsnStationV1Parm: p,
		NetworkReg:        ne,
		StationReg:        st,
		ChannelReg:        ch,
		LocationReg:       lo,
	}

	s.LevelValue, err = levelValue(p.Level)
	if err != nil {
		return s, err
	}

	// geometry bounds checking
	if p.MinLatitude != math.MaxFloat64 && p.MinLatitude < -90.0 {
		err = fmt.Errorf("minlatitude < -90.0: %f", p.MinLatitude)
		return s, err
	}

	if p.MaxLatitude != math.MaxFloat64 && p.MaxLatitude > 90.0 {
		err = fmt.Errorf("maxlatitude > 90.0: %f", p.MaxLatitude)
		return s, err
	}

	if p.MinLongitude != math.MaxFloat64 && p.MinLongitude < -180.0 {
		err = fmt.Errorf("minlongitude < -180.0: %f", p.MinLongitude)
		return s, err
	}

	if p.MaxLongitude != math.MaxFloat64 && p.MaxLongitude > 180.0 {
		err = fmt.Errorf("maxlongitude > 180.0: %f", p.MaxLongitude)
		return s, err
	}

	// Now validate longitude, latitude, and radius
	if p.Longitude != math.MaxFloat64 || p.Latitude != math.MaxFloat64 {
		if p.Longitude == math.MaxFloat64 || p.Latitude == math.MaxFloat64 {
			err = fmt.Errorf("parameter latitude and longitude must both present.")
			return s, err
		}

		if p.Longitude > 180.0 || p.Longitude < -180.0 {
			err = fmt.Errorf("invalid longitude value: %f", p.Longitude)
			return s, err
		}

		if p.Latitude > 90.0 || p.Latitude < -90.0 {
			err = fmt.Errorf("invalid latitude value: %f", p.Latitude)
			return s, err
		}

		if p.MaxRadius < 0 || p.MaxRadius > 180.0 {
			err = fmt.Errorf("invalid maxradius value.")
			return s, err
		}

		if p.MinRadius < 0 || p.MinRadius > 180.0 {
			err = fmt.Errorf("invalid minradius value.")
			return s, err
		}

		if p.MinRadius > p.MaxRadius {
			err = fmt.Errorf("minradius or maxradius range error.")
			return s, err
		}
	}

	return s, err
}

func fdsnStationVersion(r *http.Request, h http.Header, b *bytes.Buffer) error {
	err := weft.CheckQuery(r, []string{"GET"}, []string{}, []string{})
	if err != nil {
		return err
	}

	h.Set("Content-Type", "text/plain")
	_, err = b.WriteString("1.1")

	return err
}

func fdsnStationWadl(r *http.Request, h http.Header, b *bytes.Buffer) error {
	err := weft.CheckQuery(r, []string{"GET"}, []string{}, []string{})
	if err != nil {
		return err
	}

	h.Set("Content-Type", "application/xml")
	_, err = b.Write(fdsnStationWadlFile)

	return err
}

func fdsnStationV1Index(r *http.Request, h http.Header, b *bytes.Buffer) error {
	err := weft.CheckQuery(r, []string{"GET"}, []string{}, []string{})
	if err != nil {
		return err
	}

	h.Set("Content-Type", "text/html")
	_, err = b.Write(fdsnStationIndex)

	return err
}

func fdsnStationV1Handler(r *http.Request, h http.Header, b *bytes.Buffer) error {
	var v url.Values
	var params []fdsnStationV1Search

	tm := time.Now()

	switch r.Method {
	case "GET":
		v = r.URL.Query()
		p, err := parseStationV1(v)
		if err != nil {
			return fdsnError{StatusError: weft.StatusError{Code: http.StatusBadRequest, Err: err}, timestamp: tm, url: r.URL.String()}
		}
		params = []fdsnStationV1Search{p}
	case "POST":
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return fdsnError{StatusError: weft.StatusError{Code: http.StatusBadRequest, Err: err}, timestamp: tm, url: r.URL.String()}
		}
		params, err = parseStationV1Post(string(body))
		if err != nil {
			return fdsnError{StatusError: weft.StatusError{Code: http.StatusBadRequest, Err: err}, timestamp: tm, url: r.URL.String()}
		}
		if len(params) == 0 {
			return fdsnError{StatusError: weft.StatusError{Code: http.StatusBadRequest, Err: fmt.Errorf("%s", "unable to parse post request")}, timestamp: tm, url: r.URL.String()}
		}
	default:
		return fdsnError{StatusError: weft.StatusError{Code: http.StatusMethodNotAllowed}, timestamp: tm, url: r.URL.String()}
	}

	fdsnStations.RLock()
	c := *fdsnStations.fdsn // NOTE: FDSNStationXML is a pointer in struct
	fdsnStations.RUnlock()

	hasContent := c.doFilter(params)

	if !hasContent {
		return fdsnError{StatusError: weft.StatusError{Code: params[0].NoData}, timestamp: tm, url: r.URL.String()}
	}

	if params[0].Format == "xml" {
		by, err := xml.Marshal(c)
		if err != nil {
			return err
		}
		b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
		b.Write(by)
		h.Set("Content-Type", "application/xml")
	} else {
		bb := c.marshalText(params[0].LevelValue)
		b.Write(bb.Bytes())
		h.Set("Content-Type", "text/plain")
	}

	return nil
}

func (r *FDSNStationXML) trimLevel(level int) {
	for n := 0; n < len(r.Network); n++ {
		ne := &r.Network[n]
		if level < STATION_LEVEL_STATION {
			ne.Station = nil
			continue
		}
		for s := 0; s < len(ne.Station); s++ {
			st := &ne.Station[s]
			if level < STATION_LEVEL_CHANNEL {
				st.Channel = nil
				continue
			}
			for c := 0; c < len(st.Channel); c++ {
				ch := &st.Channel[c]
				if level < STATION_LEVEL_RESPONSE {
					// ch.Response is a pointer to source struct so we want to keep the source intact
					res := &ResponseType{
						ResourceId:            ch.Response.ResourceId,
						Items:                 ch.Response.Items,
						InstrumentSensitivity: ch.Response.InstrumentSensitivity,
						InstrumentPolynomial:  ch.Response.InstrumentPolynomial,
						Stage:                 nil,
					}
					ch.Response = res
					continue
				}
			}
		}
	}
}

func (r *FDSNStationXML) marshalText(levelVal int) *bytes.Buffer {
	by := bytes.NewBuffer(nil)

	switch levelVal {
	case STATION_LEVEL_NETWORK:
		by.WriteString("#Network | Description | StartTime | EndTime | TotalStations\n")
	case STATION_LEVEL_STATION:
		by.WriteString("#Network | Station | Latitude | Longitude | Elevation | SiteName | StartTime | EndTime\n")
	case STATION_LEVEL_CHANNEL:
		by.WriteString("#Network | Station | Location | Channel | Latitude | Longitude | Elevation | Depth | Azimuth | Dip | SensorDescription | Scale | ScaleFreq | ScaleUnits | SampleRate | StartTime | EndTime\n")
		// RESPONSE is not supported
	}

	for n := 0; n < len(r.Network); n++ {
		net := &r.Network[n]
		if levelVal == STATION_LEVEL_NETWORK {
			by.WriteString(fmt.Sprintf("%s|%s|%s|%s|%d\n",
				net.Code, net.Description,
				net.StartDate.MarshalFormatText(), net.EndDate.MarshalFormatText(),
				net.TotalNumberStations))
		} else {
			if levelVal == STATION_LEVEL_STATION && len(net.Station) == 0 {
				// Write Network name only
				by.WriteString(fmt.Sprintf("%s|||||||\n", net.Code))
			}
			for s := 0; s < len(net.Station); s++ {
				sta := &net.Station[s]
				if levelVal == STATION_LEVEL_STATION {
					by.WriteString(fmt.Sprintf("%s|%s|%f|%f|%f|%s|%s|%s\n",
						net.Code, sta.Code,
						sta.Latitude.Value, sta.Longitude.Value, sta.Elevation.Value,
						sta.Site.Name, sta.StartDate.MarshalFormatText(), sta.EndDate.MarshalFormatText()))
				} else {
					if len(sta.Channel) == 0 {
						// Write Station name only
						by.WriteString(fmt.Sprintf("%s|%s|||||||||||||||\n", net.Code, sta.Code))
					}
					for c := 0; c < len(sta.Channel); c++ {
						cha := &sta.Channel[c]
						var frequency float64
						if cha.Response.InstrumentSensitivity.Frequency != nil {
							frequency = *cha.Response.InstrumentSensitivity.Frequency
						}

						by.WriteString(fmt.Sprintf("%s|%s|%s|%s|%f|%f|%f|%f|%f|%f|%s|%f|%f|%s|%f|%s|%s\n",
							net.Code, sta.Code, cha.LocationCode, cha.Code,
							cha.Latitude.Value, cha.Longitude.Value, cha.Elevation.Value,
							cha.Depth.Value, cha.Azimuth.Value, cha.Dip.Value,
							cha.Sensor.Type,
							cha.Response.InstrumentSensitivity.Value,
							frequency,
							cha.Response.InstrumentSensitivity.InputUnits.Name,
							cha.SampleRate.Value,
							cha.StartDate.MarshalFormatText(), cha.EndDate.MarshalFormatText()))

					}
				}
			}
		}
	}
	return by
}

func (r *FDSNStationXML) doFilter(params []fdsnStationV1Search) bool {
	resultNetworks := make([]NetworkType, 0)
	for _, n := range r.Network {
		if n.doFilter(params) {
			resultNetworks = append(resultNetworks, n)
		}
	}

	r.Network = resultNetworks

	if len(resultNetworks) == 0 {
		return false
	}

	// Then trim the tree to the level specified in parameter before marshaling.
	// (Note: all params have the same level so I'm taking the first param's level.)
	r.trimLevel(params[0].LevelValue)

	return true
}

// For each node, check if its attribute meets how many query criterion.
// If this node meets at least one criteria, then we pass all the met criterion to next level.

func (n *NetworkType) doFilter(params []fdsnStationV1Search) bool {
	n.TotalNumberStations = len(n.Station)
	matchedParams := make([]fdsnStationV1Search, 0)
	resultStations := make([]StationType, 0)

	for _, p := range params {
		if !p.validStartEnd(time.Time(n.StartDate), time.Time(n.EndDate), STATION_LEVEL_NETWORK) {
			continue
		}
		if p.NetworkReg != nil && !matchAnyRegex(n.Code, p.NetworkReg) {
			continue
		}
		matchedParams = append(matchedParams, p)
	}

	if len(matchedParams) == 0 {
		// No match for any criterion, skip this node
		return false
	}

	if len(n.Station) == 0 {
		// for network nodes without children (though unlikely to happen):
		//   If there are query parameters for furthur levels, and there's no "*" (match anything) in the parameters,
		//     then it'll be impossible to find a match (because we don't have children)
		for _, p := range matchedParams {
			if p.StationReg != nil && !contains(p.StationReg, REGEX_ANYTHING) {
				return false
			}
			if p.ChannelReg != nil && !contains(p.ChannelReg, REGEX_ANYTHING) {
				return false
			}
			if p.LocationReg != nil && !contains(p.LocationReg, REGEX_ANYTHING) {
				return false
			}
		}
		return true
	}

	for _, s := range n.Station {
		if s.doFilter(matchedParams) {
			resultStations = append(resultStations, s)
		}
	}

	n.SelectedNumberStations = len(resultStations)
	n.Station = resultStations

	// this node only valid if the children matches any query
	return n.SelectedNumberStations > 0
}

func (s *StationType) doFilter(params []fdsnStationV1Search) bool {
	s.TotalNumberChannels = len(s.Channel)
	resultChannels := make([]ChannelType, 0)

	matchedParams := make([]fdsnStationV1Search, 0)

	for _, p := range params {
		if !p.validStartEnd(time.Time(s.StartDate), time.Time(s.EndDate), STATION_LEVEL_STATION) {
			continue
		}
		if p.StationReg != nil && !matchAnyRegex(s.Code, p.StationReg) {
			continue
		}
		if !p.validLatLng(s.Latitude, s.Longitude) {
			continue
		}
		if !p.validBounding(s.Latitude, s.Longitude) {
			continue
		}
		matchedParams = append(matchedParams, p)
	}

	if len(matchedParams) == 0 {
		// No match for any criterion, skip this node
		return false
	}

	if len(s.Channel) == 0 {
		// for station nodes without children:
		//   If there are query parameters for furthur levels, and there's no "*" (match anything) in the parameters,
		//     then it'll be impossible to find a match (because we don't have children)
		for _, p := range matchedParams {
			if p.ChannelReg != nil && !contains(p.ChannelReg, REGEX_ANYTHING) {
				return false
			}
			if p.LocationReg != nil && !contains(p.LocationReg, REGEX_ANYTHING) {
				return false
			}
		}
		return true
	}

	for _, c := range s.Channel {
		if c.doFilter(matchedParams) {
			resultChannels = append(resultChannels, c)
		}
	}

	s.SelectedNumberChannels = len(resultChannels)
	s.Channel = resultChannels

	// this node only valid if the children matches any query
	return s.SelectedNumberChannels > 0
}

func (c *ChannelType) doFilter(params []fdsnStationV1Search) bool {
	for _, p := range params {
		if !p.validStartEnd(time.Time(c.StartDate), time.Time(c.EndDate), STATION_LEVEL_CHANNEL) {
			continue
		}
		if p.ChannelReg != nil && !matchAnyRegex(c.Code, p.ChannelReg) {
			continue
		}
		if p.LocationReg != nil && !matchAnyRegex(c.LocationCode, p.LocationReg) {
			continue
		}
		if !p.validLatLng(c.Latitude, c.Longitude) {
			continue
		}
		if !p.validBounding(c.Latitude, c.Longitude) {
			continue
		}

		return true
	}

	return false
}

func (v fdsnStationV1Search) validStartEnd(start, end time.Time, level int) bool {
	/**
	 * #GeoNet/tickets/issues/4793
	 * basically apply time filter for the intended level only, because
	 * 1. network, stations and channels have different time scope, a time filter intended for station/channel/response
	 *    is not supposed to apply on network/stations,
	 * 2. while chanel and response have the same time scope (response doesn't have time attributes at all)
	 *    a time filter specified for response level need to apply on the parent channel element
	 */
	if v.LevelValue != level && (level == STATION_LEVEL_NETWORK || level == STATION_LEVEL_STATION) {
		// skip time filter
		return true
	}
	// For start/end, the "no-value" could be "0001-01-01T00:00:00" or "9999-01-01T00:00:00"
	if v.StartTime.Time != zeroDateTime {
		switch v.startMode {
		case ONBEFOREEND: // startTime
			if !end.IsZero() && end.Before(v.StartTime.Time) {
				return false
			}
		case BEFORE: // startBefore
			if !start.Equal(emptyDateTime) && !start.Before(v.StartTime.Time) {
				return false
			}
		case AFTER: // startAfter
			if start.Equal(emptyDateTime) || !start.After(v.StartTime.Time) {
				return false
			}
		}
	}

	if v.EndTime.Time != emptyDateTime {
		switch v.endMode {
		case ONAFTERSTART: // endTime
			if !start.Equal(emptyDateTime) && start.After(v.EndTime.Time) {
				return false
			}
		case BEFORE: // endBefore
			if end.IsZero() || !end.Before(v.EndTime.Time) {
				return false
			}
		case AFTER: // endAfter
			if !end.IsZero() && !end.After(v.EndTime.Time) {
				return false
			}
		}
	}
	return true
}

func (v fdsnStationV1Search) validLatLng(latitude *LatitudeType, longitude *LongitudeType) bool {
	if v.MinLatitude != math.MaxFloat64 && (latitude == nil || latitude.Value < v.MinLatitude) {
		// request to check latitude:
		// 1. this node doesn't have latitude -> check failed
		// 2. the value fall outside the range -> check failed
		// (similar logics apply for cases below)
		return false
	}

	if v.MaxLatitude != math.MaxFloat64 && (latitude == nil || latitude.Value > v.MaxLatitude) {
		return false
	}

	if v.MinLongitude != math.MaxFloat64 && (longitude == nil || longitude.Value < v.MinLongitude) {
		return false
	}

	if v.MaxLongitude != math.MaxFloat64 && (longitude == nil || longitude.Value > v.MaxLongitude) {
		return false
	}

	return true
}

func (v fdsnStationV1Search) validBounding(latitude *LatitudeType, longitude *LongitudeType) bool {
	if v.Latitude == math.MaxFloat64 {
		// not using bounding circle
		return true
	}
	if latitude == nil || longitude == nil {
		// requested bounding circle, but this node doesn't have lat/lon
		return false
	}
	d, _, err := wgs84.DistanceBearing(v.Latitude, v.Longitude, latitude.Value, longitude.Value)
	if err != nil {
		log.Printf("Error checking bounding:%s\n", err.Error())
		return false
	}
	d = d / NZ_KM_DEGREE

	if d < v.MinRadius {
		return false
	}
	if d > v.MaxRadius {
		return false
	}

	return true
}

// Download station XML from S3
func downloadStationXML(since time.Time) (by *bytes.Buffer, modified time.Time, err error) {
	s3Client, err := s3.NewWithMaxRetries(100)
	if err != nil {
		return
	}

	tp, err := s3Client.LastModified(s3Bucket, s3Meta, "")
	if err != nil {
		return
	}

	if !tp.After(since) {
		return nil, zeroDateTime, errNotModified
	}

	log.Println("Downloading fdsn station xml file from S3: ", s3Bucket+"/"+s3Meta)

	by = bytes.NewBuffer(nil)
	err = s3Client.Get(s3Bucket, s3Meta, "", by)
	if err != nil {
		return
	}

	modified = tp
	log.Println("Download complete.")
	return
}

func loadStationXML(by *bytes.Buffer, modified time.Time) (stationObj fdsnStationObj, err error) {
	var f FDSNStationXML
	if err = xml.Unmarshal(by.Bytes(), &f); err != nil {
		return
	}

	// Precondition: There's at least 1 network in the source XML.
	// Else program will crash here.
	log.Printf("Done loading %d or more stations.\n", len(f.Network[0].Station))

	stationObj.modified = modified
	stationObj.fdsn = &f

	return
}

// Periodically update data source
func setupStationXMLUpdater() {
	s := os.Getenv("STATION_RELOAD_INTERVAL")
	reloadInterval, err := strconv.Atoi(s)
	if err != nil {
		log.Printf("Warning: invalid STATION_RELOAD_INTERVAL env variable, use default value %d instead.\n", DEFAULT_RELOAD_INTERVAL)
		reloadInterval = DEFAULT_RELOAD_INTERVAL
	}
	ticker := time.NewTicker(time.Duration(reloadInterval) * time.Second)
	go func() {
		for range ticker.C {
			by, s3Modified, err := downloadStationXML(fdsnStations.modified)
			switch err {
			case errNotModified:
			// Do nothing
			case nil:
				newStations, err := loadStationXML(by, s3Modified)
				if err != nil {
					// errNotModified will be silent
					if err != errNotModified {
						log.Println("Error updating data source:", err)
					}
				} else {
					fdsnStations.Lock()
					fdsnStations.fdsn = newStations.fdsn
					fdsnStations.modified = newStations.modified
					fdsnStations.Unlock()
					log.Println("Data source updated.")
				}
			default:
				log.Println("ERROR: Download XML from S3:", err)
			}
		}
	}()
}

func levelValue(level string) (int, error) {
	switch level {
	case "station", "":
		return STATION_LEVEL_STATION, nil
	case "network":
		return STATION_LEVEL_NETWORK, nil
	case "channel":
		return STATION_LEVEL_CHANNEL, nil
	case "response":
		return STATION_LEVEL_RESPONSE, nil
	default:
		return -1, fmt.Errorf("Invalid level value.")
	}
}

func matchAnyRegex(input string, regexs []string) bool {
	for _, r := range regexs {
		if m, _ := regexp.MatchString(r, input); m {
			return true
		}
		// error here will be treated as non-matching
	}
	return false
}

// The MarshalXML funcs below use to removing output for empty date ("9999-01-01T00:00:00")
// and zero date ("0001-01-01T00:00:00")

// For XML element
func (d xsdDateTime) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if time.Time(d).Equal(zeroDateTime) || time.Time(d).Equal(emptyDateTime) {
		return nil
	}
	s, err := d.MarshalText()
	if err != nil {
		return e.EncodeElement(nil, start)
	}
	return e.EncodeElement(s, start)
}

// For attr in an XML element
func (d xsdDateTime) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	if time.Time(d).Equal(zeroDateTime) || time.Time(d).Equal(emptyDateTime) {
		return xml.Attr{}, nil
	}

	t, err := d.MarshalText()
	if err != nil {
		return xml.Attr{}, err
	}

	return xml.Attr{Name: name, Value: string(t)}, nil
}

// For format=text
func (d xsdDateTime) MarshalFormatText() string {
	if time.Time(d).Equal(zeroDateTime) || time.Time(d).Equal(emptyDateTime) {
		return ""
	}

	b, _ := d.MarshalText()
	return string(b)
}

func contains(slice []string, value string) bool {
	for _, s := range slice {
		if s == value {
			return true
		}
	}
	return false
}
