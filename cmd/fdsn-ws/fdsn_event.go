package main

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/GeoNet/fdsn/internal/fdsn"
	"github.com/GeoNet/kit/weft"
	"github.com/GeoNet/kit/wgs84"
)

var eventAbbreviations = map[string]string{
	"start":   "starttime",
	"end":     "endtime",
	"minlat":  "minlatitude",
	"maxlat":  "maxlatitude",
	"minlon":  "minlongitude",
	"maxlon":  "maxlongitude",
	"lat":     "latitude",
	"lon":     "longitude",
	"minmag":  "minmagnitude",
	"maxmag":  "maxmagnitude",
	"magtype": "magnitudetype",
}

// supported query parameters for the event service from http://www.fdsn.org/webservices/FDSN-WS-Specifications-1.1.pdf
type fdsnEventV1 struct {
	// required
	StartTime    fdsn.WsDateTime `schema:"starttime"`    // limit to events on or after the specified start time.
	EndTime      fdsn.WsDateTime `schema:"endtime"`      // limit to events on or before the specified end time.
	MinLatitude  float64         `schema:"minlatitude"`  // limit to events with a latitude larger than or equal to the specified minimum.
	MaxLatitude  float64         `schema:"maxlatitude"`  // limit to events with a latitude smaller than or equal to the specified maximum.
	MinLongitude float64         `schema:"minlongitude"` // limit to events with a longitude larger than or equal to the specified minimum.
	MaxLongitude float64         `schema:"maxlongitude"` // limit to events with a longitude smaller than or equal to the specified maximum.
	MinDepth     float64         `schema:"mindepth"`     // limit to events with depth more than the specified minimum.
	MaxDepth     float64         `schema:"maxdepth"`     // limit to events with depth less than the specified maximum.
	MinMagnitude float64         `schema:"minmagnitude"` // limit to events with a magnitude larger than the specified minimum.
	MaxMagnitude float64         `schema:"maxmagnitude"` // limit to events with a magnitude smaller than the specified maximum.
	OrderBy      string          `schema:"orderby"`      // order the result by time or magnitude with the following possibilities: time, time-asc, magnitude, magnitude-asc

	// supported optionals
	Latitude       float64         `schema:"latitude"`
	Longitude      float64         `schema:"longitude"`
	MinRadius      float64         `schema:"minradius"`
	MaxRadius      float64         `schema:"maxradius"`
	PublicID       string          `schema:"eventid"`      // select a specific event by ID; event identifiers are data center specific.
	UpdatedAfter   fdsn.WsDateTime `schema:"updatedafter"` // Limit to events updated after the specified time.
	Format         string          `schema:"format"`
	NoData         int             `schema:"nodata"` // Select status code for “no data”, either ‘204’ (default) or ‘404’.
	EventType      string          `schema:"eventtype"`
	eventTypeSlice []interface{}   // interal use only. holds matched eventtypes
}

var fdsnEventWadlFile []byte
var fdsnEventIndex []byte
var eventNotSupported = map[string]bool{
	"magnitudetype":        true,
	"limit":                true,
	"offset":               true,
	"catalog":              true,
	"contributor":          true,
	"includeallorigins":    true,
	"includeallmagnitudes": true,
	"includearrivals":      true,
}

// from https://github.com/SeisComP/common/blob/master/libs/xml/0.13/sc3ml_0.13.xsd
// https://github.com/SeisComP/common/blob/master/libs/xml/0.13/sc3ml_0.13__quakeml_1.2.xsl
var validEventTypes = []string{
	"not existing",
	"not locatable",
	"outside of network interest",
	"earthquake",
	"induced earthquake",
	"quarry blast",
	"explosion",
	"chemical explosion",
	"nuclear explosion",
	"landslide",
	"rockslide",
	"snow avalanche",
	"debris avalanche",
	"mine collapse",
	"building collapse",
	"volcanic eruption",
	"meteor impact",
	"plane crash",
	"sonic boom",
	"duplicate",
	"other",
	"not reported",
	"anthropogenic event",
	"collapse",
	"cavity collapse",
	"accidental explosion",
	"controlled explosion",
	"experimental explosion",
	"industrial explosion",
	"mining explosion",
	"road cut",
	"blasting levee",
	"induced or triggered event",
	"rock burst",
	"reservoir loading",
	"fluid injection",
	"fluid extraction",
	"crash",
	"train crash",
	"boat crash",
	"atmospheric event",
	"sonic blast",
	"acoustic noise",
	"thunder",
	"avalanche",
	"hydroacoustic event",
	"ice quake",
	"slide",
	"meteorite",
	"calving",
	"frost quake",
	"tremor pulse",
	"submarine landslide",
	"rocket launch",
	"rocket",
	"rocket impact",
	"artillery strike",
	"bomb detonation",
	"moving aircraft",
	"atmospheric meteor explosion",
	"volcano-tectonic",
	"volcanic long-period",
	"volcanic very-long-period",
	"volcanic hybrid",
	"volcanic rockfall",
	"volcanic tremor",
	"pyroclastic flow",
	"lahar",
	"other event",
}

const UNKNOWN_TYPE = "unknown"

func initEventTemplate() {
	var err error
	var b bytes.Buffer

	t, err := template.New("t").ParseFiles("assets/tmpl/fdsn-ws-event.wadl")
	if err != nil {
		log.Printf("error parsing assets/tmpl/fdsn-ws-event.wadl: %s", err.Error())
	}
	err = t.ExecuteTemplate(&b, "body", os.Getenv("HOST_CNAME"))
	if err != nil {
		log.Printf("error executing assets/tmpl/fdsn-ws-event.wadl: %s", err.Error())
	}
	fdsnEventWadlFile = b.Bytes()

	fdsnEventIndex, err = os.ReadFile("assets/fdsn-ws-event.html")
	if err != nil {
		log.Printf("error reading assets/fdsn-ws-event.html: %s", err.Error())
	}
}

func parseEventV1(v url.Values) (fdsnEventV1, error) {
	// All query parameters are optional and float zero values overlap
	// with possible request ranges so the default is set to the max float val.
	e := fdsnEventV1{
		MinLatitude:  math.MaxFloat64,
		MaxLatitude:  math.MaxFloat64,
		MinLongitude: math.MaxFloat64,
		MaxLongitude: math.MaxFloat64,
		MinDepth:     math.MaxFloat64,
		MaxDepth:     math.MaxFloat64,
		MinMagnitude: math.MaxFloat64,
		MaxMagnitude: math.MaxFloat64,
		Latitude:     math.MaxFloat64,
		Longitude:    math.MaxFloat64,
		Format:       "xml",
		MinRadius:    0.0,
		MaxRadius:    180.0,
		NoData:       204,
		EventType:    "*",
	}

	for abbrev, expanded := range eventAbbreviations {
		if val, ok := v[abbrev]; ok {
			v[expanded] = val
			delete(v, abbrev)
		}
	}

	emptyEventType := false

	for key, val := range v {
		if _, ok := eventNotSupported[key]; ok {
			return e, fmt.Errorf("\"%s\" is not supported", key)
		}
		if len(val[0]) == 0 {
			if key == "eventtype" { // eventtype allows empty value, "eventtype="
				emptyEventType = true
				continue
			}
			return e, fmt.Errorf("invalid %s value", key)
		}
	}

	err := decoder.Decode(&e, v)
	if err != nil {
		return e, err
	}

	// decoder.Decode applies default value ("*") for query like "eventtype=" (without value)
	// We overwrites it here.
	if emptyEventType {
		// overwrites default value "*"
		e.EventType = ""
	}

	if e.Format != "xml" && e.Format != "text" {
		return e, errors.New("invalid format")
	}

	if e.NoData != 204 && e.NoData != 404 {
		return e, errors.New("nodata must be 204 or 404")
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

	// Now validate longitude, latitude, and radius
	if e.Longitude != math.MaxFloat64 || e.Latitude != math.MaxFloat64 {
		if e.Longitude == math.MaxFloat64 || e.Latitude == math.MaxFloat64 {
			err = fmt.Errorf("parameter latitude and longitude must both present")
			return e, err
		}

		if e.Longitude > 180.0 || e.Longitude < -180.0 {
			err = fmt.Errorf("invalid longitude value: %f", e.Longitude)
			return e, err
		}

		if e.Latitude > 90.0 || e.Latitude < -90.0 {
			err = fmt.Errorf("invalid latitude value: %f", e.Latitude)
			return e, err
		}

		if e.MaxRadius < 0 || e.MaxRadius > 180.0 {
			err = fmt.Errorf("invalid maxradius value")
			return e, err
		}

		if e.MinRadius < 0 || e.MinRadius > 180.0 {
			err = fmt.Errorf("invalid minradius value")
			return e, err
		}

		if e.MinRadius > e.MaxRadius {
			err = fmt.Errorf("minradius or maxradius range error")
			return e, err
		}
	}

	switch e.OrderBy {
	case "", "time", "time-asc", "magnitude", "magnitude-asc":
	default:
		err = fmt.Errorf("invalid option for orderby: %s", e.OrderBy)
		return e, err
	}

	if e.EventType != "" && e.EventType != "*" {
		types := strings.Split(strings.ToLower(e.EventType), ",") // spec: case insensitive
		// we generate regexps from user's input, then check if we can match them
		regs, err := fdsn.GenRegex(types, false, true)
		if err != nil {
			err = fmt.Errorf("invalid value for eventtype: %s", e.EventType)
			return e, err
		}
		e.eventTypeSlice = make([]interface{}, 0)
		for _, t := range validEventTypes {
			if matchAnyRegex(t, regs) {
				e.eventTypeSlice = append(e.eventTypeSlice, t)
			}
		}

		if matchAnyRegex(UNKNOWN_TYPE, regs) {
			e.eventTypeSlice = append(e.eventTypeSlice, "")
		}

		if len(e.eventTypeSlice) == 0 { // the input eventtype should expect at least a match
			err = fmt.Errorf("invalid value for eventtype: %s", e.EventType)
			return e, err
		}
	} else {
		// default value, no filtering
		e.eventTypeSlice = nil
	}

	return e, nil
}

// query queries the DB for events matching e.
// The caller must close sql.Rows.
func (e *fdsnEventV1) queryQuakeML12Event() (*sql.Rows, error) {
	q := "SELECT Quakeml12Event FROM fdsn.event WHERE deleted != true"

	qq, args := e.filter()

	if qq != "" {
		q = q + " AND " + qq
	}

	switch e.OrderBy {
	case "":
	case "time":
		q += " ORDER BY origintime desc"
	case "time-asc":
		q += " ORDER BY origintime asc"
	case "magnitude":
		q += " ORDER BY magnitude desc"
	case "magnitude-asc":
		q += " ORDER BY magnitude asc"
	}

	return db.Query(q, args...)
}

func (e *fdsnEventV1) queryRaw() (*sql.Rows, error) {
	q := fmt.Sprintf("SELECT PublicID,OriginTime,Latitude,Longitude,Depth,MagnitudeType,Magnitude,COALESCE(NULLIF(EventType,''), '%s') FROM fdsn.event WHERE deleted != true", UNKNOWN_TYPE)

	qq, args := e.filter()

	if qq != "" {
		q = q + " AND " + qq
	}

	switch e.OrderBy {
	case "":
	case "time":
		q += " ORDER BY origintime desc"
	case "time-asc":
		q += " ORDER BY origintime asc"
	case "magnitude":
		q += " ORDER BY magnitude desc"
	case "magnitude-asc":
		q += " ORDER BY magnitude asc"
	}

	return db.Query(q, args...)
}

// query returns a count of events in the DB for e.
func (e *fdsnEventV1) count() (int, error) {
	q := "SELECT count(*) FROM fdsn.event WHERE deleted != true"

	qq, args := e.filter()

	if qq != "" {
		q = q + " AND " + qq
	}

	var c int
	err := db.QueryRow(q, args...).Scan(&c)
	return c, err
}

func (e *fdsnEventV1) filter() (q string, args []interface{}) {
	i := 1

	if e.PublicID != "" {
		q = fmt.Sprintf("%s publicid = $%d AND", q, i)
		args = append(args, e.PublicID)
		i++
	}

	if e.MinLatitude != math.MaxFloat64 {
		q = fmt.Sprintf("%s latitude >= $%d AND", q, i)
		args = append(args, e.MinLatitude)
		i++
	}

	if e.MaxLatitude != math.MaxFloat64 {
		q = fmt.Sprintf("%s latitude <= $%d AND", q, i)
		args = append(args, e.MaxLatitude)
		i++
	}

	if e.MinLongitude != math.MaxFloat64 {
		q = fmt.Sprintf("%s ST_X(ST_ShiftLongitude(ST_MakePoint(longitude,0.0))) >= ST_X(ST_ShiftLongitude(ST_MakePoint($%d,0.0))) AND", q, i)
		args = append(args, e.MinLongitude)
		i++
	}

	if e.MaxLongitude != math.MaxFloat64 {
		q = fmt.Sprintf("%s ST_X(ST_ShiftLongitude(ST_MakePoint(longitude,0.0))) <= ST_X(ST_ShiftLongitude(ST_MakePoint($%d,0.0))) AND", q, i)
		args = append(args, e.MaxLongitude)
		i++
	}

	if e.MinDepth != math.MaxFloat64 {
		q = fmt.Sprintf("%s depth > $%d AND", q, i)
		args = append(args, e.MinDepth)
		i++
	}

	if e.MaxDepth != math.MaxFloat64 {
		q = fmt.Sprintf("%s depth < $%d AND", q, i)
		args = append(args, e.MaxDepth)
		i++
	}

	if e.MinMagnitude != math.MaxFloat64 {
		q = fmt.Sprintf("%s magnitude > $%d AND", q, i)
		args = append(args, e.MinMagnitude)
		i++
	}

	if e.MaxMagnitude != math.MaxFloat64 {
		q = fmt.Sprintf("%s magnitude < $%d AND", q, i)
		args = append(args, e.MaxMagnitude)
		i++
	}

	if !e.StartTime.IsZero() {
		q = fmt.Sprintf("%s origintime >= $%d AND", q, i)
		args = append(args, e.StartTime.Time)
		i++
	}

	if !e.EndTime.IsZero() {
		q = fmt.Sprintf("%s origintime <= $%d AND", q, i)
		args = append(args, e.EndTime.Time)
		i++
	}

	if !e.UpdatedAfter.IsZero() {
		q = fmt.Sprintf("%s modificationtime >= $%d AND", q, i)
		args = append(args, e.UpdatedAfter.Time)
		i++
	}

	if e.MaxRadius != 180.0 {
		q = fmt.Sprintf("%s ST_Distance(ST_ShiftLongitude(origin_geom::GEOMETRY), ST_ShiftLongitude(ST_SetSRID(ST_Makepoint($%d, $%d), 4326))) <= $%d AND", q, i, i+1, i+2)
		args = append(args, e.Longitude, e.Latitude, e.MaxRadius)
		i += 3
	}

	if e.MinRadius != 0.0 {
		q = fmt.Sprintf("%s ST_Distance(ST_ShiftLongitude(origin_geom::GEOMETRY), ST_ShiftLongitude(ST_SetSRID(ST_Makepoint($%d, $%d), 4326))) >= $%d AND", q, i, i+1, i+2)
		args = append(args, e.Longitude, e.Latitude, e.MinRadius)
		i += 3
	}

	if e.eventTypeSlice != nil {
		// creating N SQL placeholders for number of matched eventTypeSlice
		p := make([]string, 0, len(e.eventTypeSlice))
		for c := range e.eventTypeSlice {
			p = append(p, fmt.Sprintf("$%d", i+c))
		}
		q = fmt.Sprintf("%s eventtype IN (%s) AND", q, strings.Join(p, ",")) // example: IN ($3,$4,$5,$6)
		args = append(args, e.eventTypeSlice...)
		i += len(e.eventTypeSlice) // nolint:ineffassign
	}

	q = strings.TrimSuffix(q, " AND")

	return
}

/*
eventV1Handler assembles QuakeML event fragments from the DB into a complete
QuakeML event.  The result set is limited to 10,000 events which will be ~1.2GB.
*/
func fdsnEventV1Handler(r *http.Request, h http.Header, b *bytes.Buffer) error {
	tm := time.Now()

	if r.Method != "GET" {
		return fdsnError{StatusError: weft.StatusError{Code: http.StatusMethodNotAllowed}, url: r.URL.String(), timestamp: tm}
	}

	e, err := parseEventV1(r.URL.Query())
	if err != nil {
		return fdsnError{StatusError: weft.StatusError{Code: http.StatusBadRequest, Err: err}, url: r.URL.String(), timestamp: tm}
	}

	c, err := e.count()
	if err != nil {
		return fdsnError{StatusError: weft.StatusError{Code: http.StatusInternalServerError, Err: err}, url: r.URL.String(), timestamp: tm}
	}

	if c == 0 {
		return fdsnError{StatusError: weft.StatusError{Code: e.NoData}, url: r.URL.String(), timestamp: tm}
	}

	if c > 10000 {
		return fdsnError{StatusError: weft.StatusError{Code: http.StatusRequestEntityTooLarge, Err: fmt.Errorf("result to large found %d events, limit is 10,000", c)}, url: r.URL.String(), timestamp: tm}
	}

	if e.Format == "xml" {
		rows, err := e.queryQuakeML12Event()
		if err != nil {
			return fdsnError{StatusError: weft.StatusError{Code: http.StatusInternalServerError, Err: err}, url: r.URL.String(), timestamp: tm}
		}
		defer func() { _ = rows.Close() }()

		b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
	<q:quakeml xmlns:q="http://quakeml.org/xmlns/quakeml/1.2" xmlns="http://quakeml.org/xmlns/bed/1.2">
	  <eventParameters publicID="smi:nz.org.geonet/NA">`)

		var xml string

		for rows.Next() {
			err = rows.Scan(&xml)
			if err != nil {
				return fdsnError{StatusError: weft.StatusError{Code: http.StatusInternalServerError, Err: err}, url: r.URL.String(), timestamp: tm}
			}

			b.WriteString(xml)
		}

		b.WriteString(`</eventParameters></q:quakeml>`)

		h.Set("Content-Type", "application/xml")
	} else {
		rows, err := e.queryRaw()
		if err != nil {
			return fdsnError{StatusError: weft.StatusError{Code: http.StatusInternalServerError, Err: err}, url: r.URL.String(), timestamp: tm}
		}
		defer func() { _ = rows.Close() }()

		b.WriteString("#EventID | Time | Latitude | Longitude | Depth/km | Author | Catalog | Contributor | ContributorID | MagType | Magnitude | MagAuthor | EventLocationName | EventType\n")

		var eventID, magType, eventType string
		var tm time.Time
		var latitude, longitude, depth, magnitude float64
		for rows.Next() {
			err = rows.Scan(&eventID, &tm, &latitude, &longitude, &depth, &magType, &magnitude, &eventType)
			if err != nil {
				return fdsnError{StatusError: weft.StatusError{Code: http.StatusInternalServerError, Err: err}, url: r.URL.String(), timestamp: tm}
			}
			loc := ""
			if l, err := wgs84.ClosestNZ(latitude, longitude); err == nil {
				loc = l.Description()
			}
			s := fmt.Sprintf("%s|%s|%.3f|%.3f|%.1f|GNS|GNS|GNS|%s|%s|%.1f|GNS|%s|%s\n", eventID, tm.UTC().Format(fdsn.WsMarshalTimeFormat), latitude, longitude, depth, eventID, magType, magnitude, loc, eventType)
			b.WriteString(s)
		}

		h.Set("Content-Type", "text/plain")
	}

	log.Printf("%s found %d events, result size %.1f (MB)", r.RequestURI, c, float64(b.Len())/1000000.0)

	return nil
}

func fdsnEventVersion(r *http.Request, h http.Header, b *bytes.Buffer) error {
	err := weft.CheckQuery(r, []string{"GET"}, []string{}, []string{})
	if err != nil {
		return err
	}

	h.Set("Content-Type", "text/plain")
	_, err = b.WriteString(eventVersion)

	return err
}

func fdsnEventContributors(r *http.Request, h http.Header, b *bytes.Buffer) error {
	err := weft.CheckQuery(r, []string{"GET"}, []string{}, []string{})
	if err != nil {
		return err
	}

	h.Set("Content-Type", "application/xml")
	_, err = b.WriteString(`<Contributors><Contributor>WEL</Contributor></Contributors>`)

	return err
}

func fdsnEventCatalogs(r *http.Request, h http.Header, b *bytes.Buffer) error {
	err := weft.CheckQuery(r, []string{"GET"}, []string{}, []string{})
	if err != nil {
		return err
	}

	h.Set("Content-Type", "application/xml")
	_, err = b.WriteString(`<Catalogs><Catalog>GeoNet</Catalog></Catalogs>`)

	return err
}

func fdsnEventWadl(r *http.Request, h http.Header, b *bytes.Buffer) error {
	err := weft.CheckQuery(r, []string{"GET"}, []string{}, []string{})
	if err != nil {
		return err
	}

	h.Set("Content-Type", "application/xml")
	_, err = b.Write(fdsnEventWadlFile)

	return err
}

func fdsnEventV1Index(r *http.Request, h http.Header, b *bytes.Buffer) error {
	err := weft.CheckQuery(r, []string{"GET"}, []string{}, []string{})
	if err != nil {
		return err
	}

	h.Set("Content-Type", "text/html")
	_, err = b.Write(fdsnEventIndex)

	return err
}
