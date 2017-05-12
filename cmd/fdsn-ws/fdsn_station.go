package main

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"fmt"
	"github.com/GeoNet/fdsn/internal/kit/s3"
	"github.com/GeoNet/weft"
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
	"time"
)

const (
	STATION_LEVEL_NETWORK   = 0
	STATION_LEVEL_STATION   = 1
	STATION_LEVEL_CHANNEL   = 2
	STATION_LEVEL_RESPONSE  = 3
	DEFAULT_RELOAD_INTERVAL = 300
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
}

// supported query parameters for the station service from http://www.fdsn.org/webservices/FDSN-WS-Specifications-1.1.pdf
type fdsnStationV1Parm struct {
	StartTime    Time    `schema:"starttime"`    // Limit to metadata epochs starting on or after the specified start time.
	EndTime      Time    `schema:"endtime"`      // Limit to metadata epochs ending on or before the specified end time.
	Network      string  `schema:"network"`      // Select one or more network codes. Can be SEED network codes or data center defined codes. Multiple codes are comma-separated.
	Station      string  `schema:"station"`      // Select one or more SEED station codes. Multiple codes are comma-separated.
	Location     string  `schema:"location"`     // Select one or more SEED location identifiers. Multiple identifiers are comma- separated. As a special case “--“ (two dashes) will be translated to a string of two space characters to match blank location IDs.
	Channel      string  `schema:"channel"`      // Select one or more SEED channel codes. Multiple codes are comma-separated.
	MinLatitude  float64 `schema:"minlatitude"`  // Limit to stations with a latitude larger than or equal to the specified minimum.
	MaxLatitude  float64 `schema:"maxlatitude"`  // Limit to stations with a latitude smaller than or equal to the specified maximum.
	MinLongitude float64 `schema:"minlongitude"` // Limit to stations with a longitude larger than or equal to the specified minimum.
	MaxLongitude float64 `schema:"maxlongitude"` // Limit to stations with a longitude smaller than or equal to the specified maximum.
	Level        string  `schema:"level"`        // Specify the level of detail for the results.
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
	fdsnStationWadlFile, err = ioutil.ReadFile("assets/fdsn-ws-station.wadl")
	if err != nil {
		log.Printf("error reading assets/fdsn-ws-station.wadl: %s", err.Error())
	}

	fdsnStationIndex, err = ioutil.ReadFile("assets/fdsn-ws-station.html")
	if err != nil {
		log.Printf("error reading assets/fdsn-ws-station.html: %s", err.Error())
	}

	emptyDateTime = time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)

	s3Bucket = os.Getenv("STATION_XML_BUCKET")
	s3Meta = os.Getenv("STATION_XML_META_KEY")
}

func parseStationV1Post(body string) ([]fdsnStationV1Search, error) {
	ret := []fdsnStationV1Search{}
	level := "station"
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

			p, err := parseStationV1(v)
			if err != nil {
				return ret, err
			}
			ret = append(ret, p)
		} else {
			return ret, fmt.Errorf("Invalid query format (POST).")
		}
	}

	return ret, nil
}

func parseStationV1(v url.Values) (fdsnStationV1Search, error) {
	// All query parameters are optional and float zero values overlap
	// with possible request ranges so the default is set to the max float val.
	p := fdsnStationV1Parm{
		MinLatitude:  math.MaxFloat64,
		MaxLatitude:  math.MaxFloat64,
		MinLongitude: math.MaxFloat64,
		MaxLongitude: math.MaxFloat64,
		Level:        "station",
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
	for _, pv := range []string{"network", "station", "channel", "location"} {
		va, ok := v[pv]
		if ok && len(va[0]) == 0 {
			return fdsnStationV1Search{}, fmt.Errorf("Invalid %s value", pv)
		}
	}

	err := decoder.Decode(&p, v)
	if err != nil {
		return fdsnStationV1Search{}, err
	}

	s := fdsnStationV1Search{
		fdsnStationV1Parm: p,
		NetworkReg:        genRegex(p.Network, false),
		StationReg:        genRegex(p.Station, false),
		ChannelReg:        genRegex(p.Channel, true),
		LocationReg:       genRegex(p.Location, false),
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

	return s, err
}

func fdsnStationVersion(r *http.Request, h http.Header, b *bytes.Buffer) *weft.Result {
	switch r.Method {
	case "GET":
		if res := weft.CheckQuery(r, []string{}, []string{}); !res.Ok {
			return res
		}

		h.Set("Content-Type", "text/plain")
		b.WriteString("1.1")
		return &weft.StatusOK
	default:
		return &weft.MethodNotAllowed
	}
}

func fdsnStationWadl(r *http.Request, h http.Header, b *bytes.Buffer) *weft.Result {
	switch r.Method {
	case "GET":
		if res := weft.CheckQuery(r, []string{}, []string{}); !res.Ok {
			return res
		}

		h.Set("Content-Type", "application/xml")
		b.Write(fdsnStationWadlFile)
		return &weft.StatusOK
	default:
		return &weft.MethodNotAllowed
	}
}

func fdsnStationV1Index(r *http.Request, h http.Header, b *bytes.Buffer) *weft.Result {
	switch r.Method {
	case "GET":
		if res := weft.CheckQuery(r, []string{}, []string{}); !res.Ok {
			return res
		}

		h.Set("Content-Type", "text/html")
		b.Write(fdsnStationIndex)
		return &weft.StatusOK
	default:
		return &weft.MethodNotAllowed
	}
}

func fdsnStationV1Handler(r *http.Request, h http.Header, b *bytes.Buffer) *weft.Result {
	var err error
	var v url.Values
	var params []fdsnStationV1Search

	switch r.Method {
	case "GET":
		v = r.URL.Query()
		p, err := parseStationV1(v)
		if err != nil {
			return weft.BadRequest(err.Error())
		}
		params = []fdsnStationV1Search{p}
	case "POST":
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return weft.BadRequest(err.Error())
		}
		params, err = parseStationV1Post(string(body))
		if err != nil {
			return weft.BadRequest(err.Error())
		}
	default:
		return &weft.MethodNotAllowed
	}

	fdsnStations.RLock()
	c := *fdsnStations.fdsn // NOTE: FDSNStationXML is a pointer in struct
	fdsnStations.RUnlock()

	hasContent := c.doFilter(params)

	if !hasContent {
		return &weft.Result{Ok: true, Code: http.StatusNoContent, Msg: ""}
	}

	// Then trim the tree to the level specified in parameter before marshaling.
	// (Note: all params have the same level so I'm taking the first param's level.)
	c.trimLevel(params[0].LevelValue)

	by, err := xml.Marshal(c)
	if err != nil {
		return weft.ServiceUnavailableError(err)
	}

	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	b.Write(by)
	h.Set("Content-Type", "application/xml")

	log.Println("Serving", r.URL.String(), len(by), "bytes")

	return &weft.StatusOK
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
					ch.Response.Stage = nil // Masking response only stops displaying "Stage"s
					continue
				}
			}
		}
	}
}

func (r *FDSNStationXML) doFilter(params []fdsnStationV1Search) bool {
	ns := make([]NetworkType, 0)

	for _, n := range r.Network {
		if n.doFilter(params) {
			ns = append(ns, n)
		}
	}

	r.Network = ns

	if len(ns) == 0 {
		// No result ( no "Network" node )
		return false
	}

	return true
}

// For each node, check if its attribute meets how many query criterion.
// If this node meets at least one criteria, then we pass all the met criterion to next level.

func (n *NetworkType) doFilter(params []fdsnStationV1Search) bool {
	n.TotalNumberStations = len(n.Station)
	matchedParams := make([]fdsnStationV1Search, 0)
	ss := make([]StationType, 0)

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

	for _, s := range n.Station {
		if s.doFilter(matchedParams) {
			ss = append(ss, s)
		}
	}

	// Special case: when requested level is deeper than this level,
	// but no child node from this node, then we should skip this node.
	if params[0].LevelValue > STATION_LEVEL_NETWORK && len(ss) == 0 {
		return false
	}

	n.SelectedNumberStations = len(ss)
	n.Station = ss

	return true
}

func (s *StationType) doFilter(params []fdsnStationV1Search) bool {
	s.TotalNumberChannels = len(s.Channel)
	cs := make([]ChannelType, 0)

	matchedParams := make([]fdsnStationV1Search, 0)

	for _, p := range params {
		if !p.validStartEnd(time.Time(s.StartDate), time.Time(s.EndDate), STATION_LEVEL_STATION) {
			continue
		}
		if p.StationReg != nil && !matchAnyRegex(s.Code, p.StationReg) {
			continue
		}
		if !p.validLatLng(s.Latitude.Value, s.Longitude.Value) {
			continue
		}

		matchedParams = append(matchedParams, p)
	}

	if len(matchedParams) == 0 {
		// No match for any criterion, skip this node
		return false
	}

	for _, c := range s.Channel {
		if c.doFilter(matchedParams) {
			cs = append(cs, c)
		}
	}

	//Special case: when requested level is deeper than this level,
	//but no child node from this node, then we should skip this node.
	if params[0].LevelValue > STATION_LEVEL_STATION && len(cs) == 0 {
		return false
	}

	s.SelectedNumberChannels = len(cs)
	s.Channel = cs

	return true
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
		if !p.validLatLng(c.Latitude.Value, c.Longitude.Value) {
			continue
		}
		return true
	}

	return false
}

func (v fdsnStationV1Search) validStartEnd(start, end time.Time, level int) bool {
	if v.LevelValue != level { // we only check for same level
		return true
	}

	if !v.StartTime.Time.IsZero() && !start.IsZero() && time.Time(start).Before(v.StartTime.Time) {
		return false
	}

	if !v.EndTime.Time.IsZero() && !end.IsZero() && time.Time(end).After(v.EndTime.Time) {
		return false
	}
	return true
}

func (v fdsnStationV1Search) validLatLng(latitude, longitude float64) bool {
	if v.MinLatitude != math.MaxFloat64 && latitude < v.MinLatitude {
		return false
	}

	if v.MaxLatitude != math.MaxFloat64 && latitude > v.MaxLatitude {
		return false
	}

	if v.MinLongitude != math.MaxFloat64 && longitude < v.MinLongitude {
		return false
	}

	if v.MaxLongitude != math.MaxFloat64 && longitude > v.MaxLongitude {
		return false
	}

	return true
}

// Download station XML from S3
func downloadStationXML(since time.Time) error {
	var by bytes.Buffer
	var err error

	var s3Client s3.S3
	s3Client, err = s3.New(100)
	if err != nil {
		return err
	}

	s3Modified, err := s3Client.LastModified(s3Bucket, s3Meta, "")
	if err != nil {
		return err
	}

	if !s3Modified.After(since) {
		return errNotModified
	}

	log.Println("Downloading fdsn station xml file from S3: ", s3Bucket+"/"+s3Meta)

	err = s3Client.Get(s3Bucket, s3Meta, "", &by)
	if err != nil {
		return err
	}

	// save it to the same file name
	if err = ioutil.WriteFile("etc/"+s3Meta, by.Bytes(), 0644); err != nil {
		return err
	}

	// Adjust the saved file's to sync with the file in S3
	if err = os.Chtimes("etc/"+s3Meta, *s3Modified, *s3Modified); err != nil {
		return err
	}

	log.Println("Download complete.")
	return nil
}

func loadStationXML(since time.Time) (stationObj fdsnStationObj, err error) {
	var b []byte
	var stat os.FileInfo

	if stat, err = os.Stat("etc/" + s3Meta); err != nil {
		return
	}

	if !stat.ModTime().After(since) {
		err = errNotModified
		return
	}

	log.Println("Loading fdsn station xml file ", "etc/"+s3Meta)
	if b, err = ioutil.ReadFile("etc/" + s3Meta); err != nil {
		return
	}

	var f FDSNStationXML
	if err = xml.Unmarshal(b, &f); err != nil {
		return
	}

	// Precondition: There's at least 1 network in the source XML.
	// Else program will crash here.
	log.Printf("Done loading %d or more stations.\n", len(f.Network[0].Station))

	stationObj.modified = stat.ModTime()
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
			err = downloadStationXML(fdsnStations.modified)
			switch err {
			case errNotModified:
				// Do nothing
			case nil:
				newStations, err := loadStationXML(fdsnStations.modified)
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

func genRegex(input string, emptyDash bool) []string {
	if len(input) == 0 {
		return nil
	}

	tokens := strings.Split(input, ",")
	result := make([]string, len(tokens))

	for i, s := range tokens {
		// turn "EH*" into "^EH.*$"
		if emptyDash && s == "--" {
			// "--" represents empty
			result[i] = "^\\s\\s$"
		} else {
			s = strings.Replace(s, "*", ".*", -1)
			s = strings.Replace(s, "?", ".?", -1)
			result[i] = "^" + s + "$"
		}
	}

	return result
}

func matchAnyRegex(input string, regexs []string) bool {
	for _, r := range regexs {
		if m, _ := regexp.MatchString(r, input); m == true {
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
	return e.EncodeElement(time.Time(d), start)
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
