package main

import (
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
	"strings"
	"time"
)

const (
	STATION_LEVEL_NETWORK = 0
	STATION_LEVEL_STATION = 1
	STATION_LEVEL_CHANNEL = 2
)

// supported query parameters for the station service from http://www.fdsn.org/webservices/FDSN-WS-Specifications-1.1.pdf
type fdsnStationV1Parm struct {
	StartTime    Time `schema:"starttime"` // Limit to metadata epochs starting on or after the specified start time.
	Start        Time
	EndTime      Time `schema:"endtime"` // Limit to metadata epochs ending on or before the specified end time.
	End          Time
	Network      string `schema:"network"` // Select one or more network codes. Can be SEED network codes or data center defined codes. Multiple codes are comma-separated.
	Net          string
	Station      string `schema:"station"` // Select one or more SEED station codes. Multiple codes are comma-separated.
	Sta          string
	Location     string `schema:"location"` // Select one or more SEED location identifiers. Multiple identifiers are comma- separated. As a special case “--“ (two dashes) will be translated to a string of two space characters to match blank location IDs.
	Loc          string
	Channel      string `schema:"channel"` // Select one or more SEED channel codes. Multiple codes are comma-separated.
	Cha          string
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

var fdsnStationWadlFile []byte
var fdsnStationIndex []byte
var fdsnStations FDSNStationXML
var stationsLoaded bool

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

	// The loading+unmarshaling complete fdsn-station xml file could take quite a long time.
	// So we're loading it (locally or download from S3) in a goroutine.
	// Before unmarshling is done the service returns 500 error with "not ready" message.
	go func() {
		s3Bucket := os.Getenv("FDSN_STATION_XML_BUCKET")
		s3MetaKey := os.Getenv("FDSN_STATION_XML_META_KEY")

		var by bytes.Buffer
		var r *bytes.Reader
		var b []byte
		var err error

		// If we've found local cache the use the cache.
		if _, err = os.Stat("etc/" + s3MetaKey); err == nil {
			log.Println("Loading fdsn station xml file from local: ", "etc/"+s3MetaKey)
			if b, err = ioutil.ReadFile("etc/" + s3MetaKey); err != nil {
				log.Println(err)
			}
			// Note: if anything goes wrong here, we download from S3 as the fallback.
		}

		if err != nil {
			log.Println("Loading fdsn station xml file from S3: ", s3Bucket, s3MetaKey)

			s3Client, err := s3.New(100)
			if err != nil {
				log.Fatalf("creating S3 client: %s", err)
			}

			err = s3Client.Get(s3Bucket, s3MetaKey, "", &by)
			if err != nil {
				log.Println(err.Error())
				return
			}

			// save it to the same file name
			if err = ioutil.WriteFile("etc/"+s3MetaKey, by.Bytes(), 0644); err != nil {
				log.Println(err.Error())
			}

			r = bytes.NewReader(by.Bytes())
			if b, err = ioutil.ReadAll(r); err != nil {
				log.Println(err.Error())
			}
		}

		if err = xml.Unmarshal(b, &fdsnStations); err != nil {
			log.Printf("error unmarshaling fdsn-station.xml", err.Error())
		} else {
			log.Println("Done loading stations:", len(fdsnStations.Network[0].Station))
			stationsLoaded = true
		}
	}()

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
	if e.Net != "" {
		e.Network = e.Net
	}
	if e.Sta != "" {
		e.Station = e.Sta
	}
	if e.Cha != "" {
		e.Channel = e.Cha
	}
	if e.Loc != "" {
		e.Location = e.Loc
	}

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

	e.LevelValue = levelValue(e.Level)

	if e.Network != "" {
		e.NetworkReg = genRegex(e.Network)
	}

	if e.Station != "" {
		e.StationReg = genRegex(e.Station)
	}

	if e.Channel != "" {
		e.ChannelReg = genRegex(e.Channel)
	}

	if e.Location != "" {
		e.LocationReg = genRegex(e.Location)
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
	if !stationsLoaded {
		return weft.ServiceUnavailableError(fmt.Errorf("Station data not ready."))
	}

	if r.Method != "GET" {
		return &weft.MethodNotAllowed
	}

	var err error
	params, err := parseStationV1(r.URL.Query())
	if err != nil {
		return weft.BadRequest(err.Error())
	}

	c := fdsnStations
	if !c.doFilter(params) {
		return &weft.Result{Ok: false, Code: http.StatusNoContent, Msg: "The query resut is empty."}
	}

	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	var by []byte

	by, err = xml.Marshal(&c)
	if err != nil {
		return weft.BadRequest(err.Error())
	}

	b.Write(by)
	h.Set("Content-Type", "application/xml")

	return &weft.StatusOK
}

func (r *FDSNStationXML) doFilter(params fdsnStationV1Parm) bool {
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

func (n *NetworkType) doFilter(params fdsnStationV1Parm) bool {
	if !params.validStartEnd(time.Time(n.StartDate), time.Time(n.EndDate), STATION_LEVEL_NETWORK) {
		return false
	}
	if params.NetworkReg != nil && !matchAnyRegex(n.Code, params.NetworkReg) {
		return false
	}

	if params.LevelValue < STATION_LEVEL_STATION {
		n.Station = nil
		return true
	}

	n.TotalNumberStations = len(n.Station)

	ss := make([]StationType, 0)
	for _, s := range n.Station {
		if s.doFilter(params) {
			ss = append(ss, s)
		}
	}

	n.SelectedNumberStations = len(ss)
	n.Station = ss

	// Special case: when requested level is deeper than this level,
	// but no child node from this node, then we should skip this node.
	if params.LevelValue > STATION_LEVEL_NETWORK && len(ss) == 0 {
		return false
	}

	return true
}

func (s *StationType) doFilter(params fdsnStationV1Parm) bool {
	if !params.validStartEnd(time.Time(s.StartDate), time.Time(s.EndDate), STATION_LEVEL_STATION) {
		return false
	}
	if params.StationReg != nil && !matchAnyRegex(s.Code, params.StationReg) {
		return false
	}
	if !params.validLatLng(s.Latitude.Double, s.Longitude.Double) {
		return false
	}
	if params.LevelValue < STATION_LEVEL_CHANNEL {
		s.Channel = nil
		return true
	}

	s.TotalNumberChannels = len(s.Channel)

	cs := make([]ChannelType, 0)
	for _, c := range s.Channel {
		if c.doFilter(params) {
			cs = append(cs, c)
		}
	}

	s.SelectedNumberChannels = len(cs)
	s.Channel = cs

	// Special case: when requested level is deeper than this level,
	// but no child node from this node, then we should skip this node.
	if params.LevelValue > STATION_LEVEL_STATION && len(cs) == 0 {
		return false
	}

	return true
}

func (c *ChannelType) doFilter(params fdsnStationV1Parm) bool {
	if !params.validStartEnd(time.Time(c.StartDate), time.Time(c.EndDate), STATION_LEVEL_CHANNEL) {
		return false
	}
	if params.ChannelReg != nil && !matchAnyRegex(c.Code, params.ChannelReg) {
		return false
	}
	if params.LocationReg != nil && !matchAnyRegex(c.LocationCode, params.LocationReg) {
		return false
	}
	if !params.validLatLng(c.Latitude.Double, c.Longitude.Double) {
		return false
	}

	return true
}

func levelValue(level string) int {
	switch level {
	case "station":
		return STATION_LEVEL_STATION
	case "network":
		return STATION_LEVEL_NETWORK
	case "channel":
		return STATION_LEVEL_CHANNEL
	default:
		return STATION_LEVEL_STATION
	}
}

func genRegex(input string) []string {
	tokens := strings.Split(input, ",")
	result := make([]string, len(tokens))

	for i, s := range tokens {
		// turn "EH*" into "^EH.*$"
		// "--" represents empty
		if s == "--" {
			result[i] = "^$"
		} else {
			result[i] = "^" + strings.Replace(s, "*", ".*", -1) + "$"
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
