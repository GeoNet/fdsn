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

// supported query parameters for the station service from http://www.fdsn.org/webservices/FDSN-WS-Specifications-1.1.pdf
type fdsnStationV1Parm struct {
	StartTime    Time `schema:"starttime"` // Limit to metadata epochs starting on or after the specified start time.
	Start        Time
	EndTime      Time `schema:"endtime"` // Limit to metadata epochs ending on or before the specified end time.
	End          Time
	Network      *string `schema:"network"` // Select one or more network codes. Can be SEED network codes or data center defined codes. Multiple codes are comma-separated.
	Net          *string
	Station      *string `schema:"station"` // Select one or more SEED station codes. Multiple codes are comma-separated.
	Sta          *string
	Location     *string `schema:"location"` // Select one or more SEED location identifiers. Multiple identifiers are comma- separated. As a special case “--“ (two dashes) will be translated to a string of two space characters to match blank location IDs.
	Loc          *string
	Channel      *string `schema:"channel"` // Select one or more SEED channel codes. Multiple codes are comma-separated.
	Cha          *string
	MinLatitude  float64 `schema:"minlatitude"` // Limit to stations with a latitude larger than or equal to the specified minimum.
	MinLat       float64
	MaxLatitude  float64 `schema:"maxlatitude"` // Limit to stations with a latitude smaller than or equal to the specified maximum.
	MaxLat       float64
	MinLongitude float64 `schema:"minlongitude"` // Limit to stations with a longitude larger than or equal to the specified minimum.
	MinLon       float64
	MaxLongitude float64 `schema:"maxlongitude"` // Limit to stations with a longitude smaller than or equal to the specified maximum.
	MaxLon       float64
	Level        string `schema:"level"` // Specify the level of detail for the results.
	LevelValue   int
	NetworkReg   []string
	StationReg   []string
	LocationReg  []string
	ChannelReg   []string
}

func (v fdsnStationV1Parm) validStartEnd(start, end time.Time, level int) bool {
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

func (v fdsnStationV1Parm) validLatLng(latitude, longitude float64) bool {
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

type fdsnStationObj struct {
	fdsn     *FDSNStationXML
	modified time.Time
	sync.RWMutex
}

var fdsnStationWadlFile []byte
var fdsnStationIndex []byte
var fdsnStations fdsnStationObj
var zeroDateTime time.Time
var emptyDateTime time.Time
var errNotModified = fmt.Errorf("Not modified.")

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

	zeroDateTime = time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)
	emptyDateTime = time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)

	fdsnStations, err = loadStationXML(time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		// We don't have to consider ERR_NOT_MODIFIED here.
		log.Fatalf("Error loading station xml: %s\n", err.Error())
	}

	s := os.Getenv("STATION_RELOAD_INTERVAL")
	reloadInterval, err := strconv.Atoi(s)
	if err != nil {
		log.Printf("Warning: invalid STATION_RELOAD_INTERVAL env variable, use default value %d instead.\n", DEFAULT_RELOAD_INTERVAL)
		reloadInterval = DEFAULT_RELOAD_INTERVAL
	}

	ticker := time.NewTicker(time.Duration(reloadInterval) * time.Second)
	go func() {
		for range ticker.C {
			newStations, err := loadStationXML(fdsnStations.modified)
			if err != nil {
				if err == errNotModified {
					log.Println("No update for data source.")
				} else {
					log.Println("Error updating data source:", err)
				}
			} else {
				fdsnStations.Lock()
				fdsnStations.fdsn = newStations.fdsn
				fdsnStations.modified = newStations.modified
				fdsnStations.Unlock()
				log.Println("Data source updated.")
			}
		}
	}()
}

func parseStationV1Post(body string) ([]fdsnStationV1Parm, error) {
	ret := []fdsnStationV1Parm{}
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

func parseStationV1(v url.Values) (fdsnStationV1Parm, error) {
	// All query parameters are optional and float zero values overlap
	// with possible request ranges so the default is set to the max float val.
	e := fdsnStationV1Parm{
		MinLatitude:  math.MaxFloat64,
		MinLat:       math.MaxFloat64,
		MaxLatitude:  math.MaxFloat64,
		MaxLat:       math.MaxFloat64,
		MinLongitude: math.MaxFloat64,
		MinLon:       math.MaxFloat64,
		MaxLongitude: math.MaxFloat64,
		MaxLon:       math.MaxFloat64,
		Level:        "station",
		LevelValue:   STATION_LEVEL_STATION,
	}

	err := decoder.Decode(&e, v)
	if err != nil {
		return e, err
	}

	// Let all abbreviations override
	if !e.Start.IsZero() {
		e.StartTime = e.Start
	}
	if !e.End.IsZero() {
		e.EndTime = e.End
	}
	if e.MinLat != math.MaxFloat64 {
		e.MinLatitude = e.MinLat
	}
	if e.MaxLat != math.MaxFloat64 {
		e.MaxLatitude = e.MaxLat
	}
	if e.MinLon != math.MaxFloat64 {
		e.MinLongitude = e.MinLon
	}
	if e.MaxLon != math.MaxFloat64 {
		e.MaxLongitude = e.MaxLon
	}
	if e.Net != nil {
		e.Network = e.Net
	}
	if e.Sta != nil {
		e.Station = e.Sta
	}
	if e.Cha != nil {
		e.Channel = e.Cha
	}
	if e.Loc != nil {
		e.Location = e.Loc
	}

	e.LevelValue, err = levelValue(e.Level)
	if err != nil {
		return e, err
	}

	e.NetworkReg = genRegex(e.Network)
	e.StationReg = genRegex(e.Station)
	e.ChannelReg = genRegex(e.Channel)
	e.LocationReg = genRegex(e.Location)

	// geometry bounds checking
	if e.MinLatitude != math.MaxFloat64 && e.MinLatitude < -90.0 {
		err = fmt.Errorf("minlatitude < -90.0: %f", e.MinLatitude)
		return e, err
	}

	if e.MaxLatitude != math.MaxFloat64 && e.MaxLatitude > 90.0 {
		err = fmt.Errorf("maxlatitude > 90.0: %f", e.MaxLatitude)
		return e, err
	}

	if e.MinLongitude != math.MaxFloat64 && e.MinLongitude < -180.0 {
		err = fmt.Errorf("minlongitude < -180.0: %f", e.MinLongitude)
		return e, err
	}

	if e.MaxLongitude != math.MaxFloat64 && e.MaxLongitude > 180.0 {
		err = fmt.Errorf("maxlongitude > 180.0: %f", e.MaxLongitude)
		return e, err
	}

	return e, err
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
	var params []fdsnStationV1Parm

	switch r.Method {
	case "GET":
		v = r.URL.Query()
		p, err := parseStationV1(v)
		if err != nil {
			return weft.BadRequest(err.Error())
		}
		params = []fdsnStationV1Parm{p}
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

	by, err := xml.Marshal(c)
	if err != nil {
		return weft.BadRequest(err.Error())
	}

	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	b.Write(by)
	h.Set("Content-Type", "application/xml")

	log.Println("Serving", r.URL.String(), len(by), "bytes")

	return &weft.StatusOK
}

func (r *FDSNStationXML) doFilter(params []fdsnStationV1Parm) bool {
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

func (n *NetworkType) doFilter(params []fdsnStationV1Parm) bool {
	n.TotalNumberStations = len(n.Station)
	matchedParams := make([]fdsnStationV1Parm, 0)
	ss := make([]StationType, 0)

	for _, p := range params {
		if !p.validStartEnd(time.Time(n.StartDate), time.Time(n.EndDate), STATION_LEVEL_NETWORK) {
			continue
		}
		if p.NetworkReg != nil && !matchAnyRegex(n.Code, p.NetworkReg) {
			continue
		}
		if p.LevelValue < STATION_LEVEL_STATION {
			n.Station = nil
			return true
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

func (s *StationType) doFilter(params []fdsnStationV1Parm) bool {
	s.TotalNumberChannels = len(s.Channel)
	cs := make([]ChannelType, 0)

	matchedParams := make([]fdsnStationV1Parm, 0)

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
		if p.LevelValue < STATION_LEVEL_CHANNEL {
			s.Channel = nil
			return true
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

	// Special case: when requested level is deeper than this level,
	// but no child node from this node, then we should skip this node.
	if params[0].LevelValue > STATION_LEVEL_STATION && len(cs) == 0 {
		return false
	}

	s.SelectedNumberChannels = len(cs)
	s.Channel = cs

	return true
}

func (c *ChannelType) doFilter(params []fdsnStationV1Parm) bool {
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

		if p.LevelValue <= STATION_LEVEL_CHANNEL {
			// The user is not asking for response, remove responses before return
			c.Response = ResponseType{}
		}
		return true
	}

	return false
}

func loadStationXML(since time.Time) (stationObj fdsnStationObj, err error) {
	bucket := os.Getenv("STATION_XML_BUCKET")
	meta := os.Getenv("STATION_XML_META_KEY")

	var by bytes.Buffer
	var r *bytes.Reader
	var b []byte
	var modified time.Time

	// If we've found local cache the use the cache.
	if stat, err := os.Stat("etc/" + meta); err == nil {
		if !stat.ModTime().After(since) {
			return stationObj, errNotModified
		}
		log.Println("Loading fdsn station xml file from local: ", "etc/"+meta)
		if b, err = ioutil.ReadFile("etc/" + meta); err != nil {
			log.Println("Warning:", err.Error())
		} else {
			modified = stat.ModTime()
		}
		// Note: if anything goes wrong here, we download from S3 as the fallback.
	}

	if err != nil {
		log.Println("Loading fdsn station xml file from S3: ", bucket, meta)

		var s3Client s3.S3
		var t *time.Time
		s3Client, err = s3.New(100)
		if err != nil {
			return
		}

		t, err = s3Client.LastModified(bucket, meta, "")
		if err != nil {
			return
		}

		if !t.After(since) {
			return stationObj, errNotModified
		}

		err = s3Client.Get(bucket, meta, "", &by)
		if err != nil {
			return
		}

		// save it to the same file name
		if err = ioutil.WriteFile("etc/"+meta, by.Bytes(), 0644); err != nil {
			return
		}

		r = bytes.NewReader(by.Bytes())
		if b, err = ioutil.ReadAll(r); err != nil {
			return
		}

		modified = *t
	}

	var f FDSNStationXML
	if err = xml.Unmarshal(b, &f); err != nil {
		return
	}

	// Precondition: There's at least 1 network in the source XML.
	// Else program will crash here.
	log.Printf("Done loading %d or more stations.\n", len(f.Network[0].Station))

	stationObj.modified = modified
	stationObj.fdsn = &f

	return
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
		return -1, fmt.Errorf("Invalid level string.")
	}
}

func genRegex(input *string) []string {
	if input == nil {
		return nil // Default value is nil. Setting nil to regex can make further check to skip.
	}

	tokens := strings.Split(*input, ",")
	result := make([]string, len(tokens))

	for i, s := range tokens {
		// turn "EH*" into "^EH.*$"
		// "--" represents empty
		if s == "--" {
			result[i] = "^$"
		} else {
			s = strings.Replace(s, "*", ".*", -1)
			s = strings.Replace(s, "?", ".?", -1)
			result[i] = "^" + s + "$"
		}
	}

	return result
}

func matchAnyRegex(input string, regexs []string) bool {
	if regexs == nil {
		return true
	}
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
