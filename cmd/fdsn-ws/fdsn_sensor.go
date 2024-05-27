package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/GeoNet/fdsn/internal/valid"
	"github.com/GeoNet/kit/aws/s3"
	"github.com/GeoNet/kit/weft"
)

type Sensor struct {
	Code        string `xml:"code,attr,omitempty" json:"code,omitempty"`
	Model       string `xml:"model,attr,omitempty" json:"model,omitempty"`
	Make        string `xml:"make,attr,omitempty" json:"make,omitempty"`
	Type        string `xml:"type,attr,omitempty" json:"type,omitempty"`
	Channels    string `xml:"channels,attr,omitempty" json:"channels,omitempty"`
	Description string `xml:"description,attr,omitempty" json:"description,omitempty"`
	Property    string `xml:"property,attr,omitempty" json:"property,omitempty"`
	Aspect      string `xml:"aspect,attr,omitempty" json:"aspect,omitempty"`

	Azimuth float64 `xml:"azimuth,attr,omitempty" json:"azimuth,omitempty"`
	Dip     float64 `xml:"dip,attr,omitempty" json:"dip,omitempty"`
	Method  string  `xml:"method,attr,omitempty" json:"method,omitempty"`

	Vertical float64 `xml:"vertical,attr,omitempty" json:"vertical,omitempty"`
	North    float64 `xml:"north,attr,omitempty" json:"north,omitempty"`
	East     float64 `xml:"east,attr,omitempty" json:"east,omitempty"`

	StartDate time.Time `xml:"startDate,attr,omitempty" json:"start-date,omitempty"`
	EndDate   time.Time `xml:"endDate,attr,omitempty" json:"end-date,omitempty"`
}

type Site struct {
	Code string `xml:"code,attr,omitempty" json:"code,omitempty"`
	Name string `xml:"name,attr,omitempty" json:"name,omitempty"`

	Latitude       float64 `xml:"latitude,attr,omitempty" json:"latitude,omitempty"`
	Longitude      float64 `xml:"longitude,attr,omitempty" json:"longitude,omitempty"`
	Elevation      float64 `xml:"elevation,attr,omitempty" json:"elevation,omitempty"`
	Depth          float64 `xml:"depth,attr,omitempty" json:"depth,omitempty"`
	Datum          string  `xml:"datum,attr,omitempty" json:"datum,omitempty"`
	Survey         string  `xml:"survey,attr,omitempty" json:"survey,omitempty"`
	RelativeHeight float64 `xml:"relativeHeight,attr,omitempty" json:"relative-height,omitempty"`

	StartDate time.Time `xml:"startDate,attr,omitempty" json:"start-date,omitempty"`
	EndDate   time.Time `xml:"endDate,attr,omitempty" json:"end-date,omitempty"`

	Sensors []Sensor `xml:"Sensor,omitempty" json:"sensor,omitempty"`
}

type Station struct {
	Code    string `xml:"code,attr" json:"code,omitempty"`
	Network string `xml:"network,attr,omitempty" json:"network,omitempty"`

	Name        string    `xml:"name,attr,omitempty" json:"name,omitempty"`
	Description string    `xml:"description,attr,omitempty" json:"description,omitempty"`
	StartDate   time.Time `xml:"startDate,attr,omitempty" json:"start-date,omitempty"`
	EndDate     time.Time `xml:"endDate,attr,omitempty" json:"end-date,omitempty"`

	Latitude  float64 `xml:"latitude,attr,omitempty" json:"latitude,omitempty"`
	Longitude float64 `xml:"longitude,attr,omitempty" json:"longitude,omitempty"`
	Elevation float64 `xml:"elevation,attr,omitempty" json:"elevation,omitempty"`
	Depth     float64 `xml:"depth,attr,omitempty" json:"depth,omitempty"`
	Datum     string  `xml:"datum,attr,omitempty" json:"datum,omitempty"`

	Sites []Site `xml:"Site,omitempty" json:"site,omitempty"`
}

type Mark struct {
	Code    string `xml:"code,attr" json:"code,omitempty"`
	Network string `xml:"network,attr,omitempty" json:"network,omitempty"`

	Name        string `xml:"name,attr,omitempty" json:"name,omitempty"`
	Description string `xml:"description,attr,omitempty" json:"description,omitempty"`
	DomesNumber string `xml:"domesNumber,attr,omitempty" json:"domes-number,omitempty"`

	Latitude           float64 `xml:"latitude,attr,omitempty" json:"latitude,omitempty"`
	Longitude          float64 `xml:"longitude,attr,omitempty" json:"longitude,omitempty"`
	Elevation          float64 `xml:"elevation,attr,omitempty" json:"elevation,omitempty"`
	GroundRelationship float64 `xml:"groundRelationship,attr,omitempty" json:"ground-relationship,omitempty"`
	Datum              string  `xml:"datum,attr,omitempty" json:"datum,omitempty"`

	MarkType        string  `xml:"markType,attr,omitempty" json:"mark-type,omitempty"`
	MonumentType    string  `xml:"monumentType,attr,omitempty" json:"monument-type,omitempty"`
	FoundationType  string  `xml:"foundationType,attr,omitempty" json:"foundation-type,omitempty"`
	FoundationDepth float64 `xml:"foundationDepth,attr,omitempty" json:"foundation-depth,omitempty"`
	Bedrock         string  `xml:"bedrock,attr,omitempty" json:"bedrock,omitempty"`
	Geology         string  `xml:"geology,attr,omitempty" json:"geology,omitempty"`

	StartDate time.Time `xml:"startDate,attr,omitempty" json:"start-date,omitempty"`
	EndDate   time.Time `xml:"endDate,attr,omitempty" json:"end-date,omitempty"`

	Antennas  []Sensor `xml:"Antenna,omitempty" json:"antenna,omitempty"`
	Receivers []Sensor `xml:"Receiver,omitempty" json:"receiver,omitempty"`
}

type View struct {
	Code        string `xml:"code,attr" json:"code,omitempty"`
	Label       string `xml:"label,attr,omitempty" json:"label,omitempty"`
	Description string `xml:"description,attr,omitempty" json:"description,omitempty"`

	Azimuth float64 `xml:"azimuth,attr,omitempty" json:"azimuth,omitempty"`
	Method  string  `xml:"method,attr,omitempty" json:"method,omitempty"`
	Dip     float64 `xml:"dip,attr,omitempty" json:"dip,omitempty"`

	StartDate time.Time `xml:"startDate,attr,omitempty" json:"start-date,omitempty"`
	EndDate   time.Time `xml:"endDate,attr,omitempty" json:"end-date,omitempty"`

	Sensors []Sensor `xml:"Sensor,omitempty" json:"sensor,omitempty"`
}

type Mount struct {
	Code    string `xml:"code,attr" json:"code,omitempty"`
	Network string `xml:"network,attr,omitempty" json:"network,omitempty"`

	Name        string `xml:"name,attr,omitempty" json:"name,omitempty"`
	Description string `xml:"description,attr,omitempty" json:"description,omitempty"`
	Mount       string `xml:"mount,attr,omitempty" json:"mount,omitempty"`

	Latitude  float64 `xml:"latitude,attr" json:"latitude,omitempty"`
	Longitude float64 `xml:"longitude,attr" json:"longitude,omitempty"`
	Elevation float64 `xml:"elevation,attr" json:"elevation,omitempty"`
	Datum     string  `xml:"datum,attr,omitempty" json:"datum,omitempty"`

	StartDate time.Time `xml:"startDate,attr,omitempty" json:"start-date,omitempty"`
	EndDate   time.Time `xml:"endDate,attr,omitempty" json:"end-date,omitempty"`

	Views []View `xml:"View,omitempty" json:"view,omitempty"`
}

type SensorStation interface {
	toJson() ([]byte, error)
}

type Group struct {
	Name string `xml:"name,omitempty,attr" json:"name,omitempty"`

	Marks    []Mark    `xml:"Mark,omitempty" json:"marks,omitempty"`
	Mounts   []Mount   `xml:"Mount,omitempty" json:"mounts,omitempty"`
	Samples  []Station `xml:"Sample,omitempty" json:"samples,omitempty"`
	Stations []Station `xml:"Station,omitempty" json:"stations,omitempty"`
}

type FDSNSensorXML struct {
	XMLName xml.Name `xml:"SensorXML"`

	Groups []Group `xml:"Group,omitempty" json:"group,omitempty"`

	Stations []Station `xml:"Station,omitempty" json:"station,omitempty"`
	Marks    []Mark    `xml:"Mark,omitempty" json:"mark,omitempty"`
	Buoys    []Station `xml:"Buoy,omitempty" json:"buoy,omitempty"`
	Mounts   []Mount   `xml:"Mount,omitempty" json:"mount,omitempty"`
	Samples  []Station `xml:"Sample,omitempty" json:"sample,omitempty"`
}

type fdsnSensorObj struct {
	sensors  *FDSNSensorXML
	modified time.Time
	sync.RWMutex
}

// A sensor is installed at a featureProperties and data recorded on a number of channels.
// For seismic sites the featureProperties is Network.Station.Location
// For GNSS sites the featureProperties is the Mark.
type featureProperties struct {
	Code       string    `json:",omitempty"` // Code for search. Set to Mark or Station Code
	Name       string    `json:",omitempty"` // Station code for seismic sites
	Location   string    `json:",omitempty"` // Location code for seismic sites
	Start      time.Time `json:",omitempty"`
	End        time.Time `json:",omitempty"` //end time, 0001-01-01 00:00:00 +0000 UTC indicates the channel is still open.
	SensorType string    `json:",omitempty"`
}

type point struct {
	Type        string     `json:"type"`
	Coordinates [2]float64 `json:"coordinates"`
}

type Feature struct {
	Type       string            `json:"type"`
	Properties featureProperties `json:"properties"`
	Geometry   point             `json:"geometry"`
}

type features []Feature

type FeatureCollection struct {
	Type     string   `json:"type"`
	Features features `json:"features"`
}

var (
	fdsnSensors    fdsnSensorObj
	s3SensorBucket string
	s3SensorMeta   string
)

var allSensorTypes = map[string]string{
	"1":  "Air pressure sensor",           //station
	"2":  "Broadband seismometer",         //station
	"3":  "Coastal sea level gauge",       //station
	"4":  "DART bottom pressure recorder", //station
	"5":  "DOAS spectrometer",             //mount
	"6":  "Environmental sensor",          //station
	"7":  "Geomagnetic sensor",            // station
	"8":  "GNSS/GPS",                      //mark
	"9":  "Lake level gauge",              //station
	"10": "Manual collection",             //sample
	"11": "Short period seismometer",      //station
	"12": "Strong motion sensor",          //station
	"13": "Volcano camera",                //mount
}

func (s Station) toJson() ([]byte, error) {
	return json.Marshal(s)
}

func (s Mark) toJson() ([]byte, error) {
	return json.Marshal(s)
}

func (s Mount) toJson() ([]byte, error) {
	return json.Marshal(s)
}

func init() {
	var err error

	emptyDateTime = time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)

	s3SensorBucket = "geonet-meta"            //os.Getenv("SENSOR_XML_BUCKET")
	s3SensorMeta = "config/sensor/sensor.xml" //os.Getenv("SENSOR_XML_META_KEY")

	by, modified, err := downloadSensorXML(zeroDateTime)
	if err != nil {
		log.Fatalf("## Download sensors from S3 error: %s\n", err.Error())
	}

	fdsnSensors, err = loadSensorXML(by, modified)
	if err != nil {
		log.Fatalf("Error loading xml: %s\n", err)
	}
}

// get stations by: sensorType(group), start/end time in geo json format
// station?sensorType=2,6&endDate=9999-01-01
func sensorStaionsHandler(r *http.Request, h http.Header, b *bytes.Buffer) error {
	err := weft.CheckQuery(r, []string{"GET"}, []string{"sensorType"}, []string{"startDate", "endDate"})
	if err != nil {
		return weft.StatusError{Code: http.StatusBadRequest, Err: err}
	}

	q := r.URL.Query()
	sensorType := q.Get("sensorType")

	start := q.Get("startDate")
	var d1 time.Time
	if start != "" { //empty as zero time
		d1, err = valid.ParseDate(start)
		if err != nil {
			return weft.StatusError{Code: http.StatusBadRequest, Err: err}
		}
	}
	d2, err := valid.ParseDate(q.Get("endDate")) //empty as now

	if err != nil {
		return weft.StatusError{Code: http.StatusBadRequest, Err: err}
	}

	fdsnStations.RLock()
	c := *fdsnSensors.sensors
	fdsnStations.RUnlock()

	stationFeatures := c.getStationFeatures(sensorType, d1, d2)
	scJason, err := json.Marshal(&stationFeatures)
	if err != nil {
		return weft.StatusError{Code: http.StatusInternalServerError}
	}
	h.Set("Content-Type", "application/vnd.geo+json")
	_, err = b.Write(scJason)
	if err != nil {
		return weft.StatusError{Code: http.StatusInternalServerError}
	}
	return nil
}

// get sensors for specified sensorType, station and location code
// fdsnws/sensor?sensorType=2&station=WEL&location=10
func sensorHandler(r *http.Request, h http.Header, b *bytes.Buffer) error {
	err := weft.CheckQuery(r, []string{"GET"}, []string{"sensorType", "station"}, []string{"location"})
	if err != nil {
		return weft.StatusError{Code: http.StatusBadRequest, Err: err}
	}

	q := r.URL.Query()
	sensorType := q.Get("sensorType")
	station := q.Get("station")
	location := q.Get("location")

	fdsnStations.RLock()
	c := *fdsnSensors.sensors
	fdsnStations.RUnlock()

	scJason := []byte("{}")
	sensors := c.getSensors(sensorType, station, location)
	if sensors != nil {
		scJason, err = sensors.toJson()
		if err != nil {
			return weft.StatusError{Code: http.StatusInternalServerError, Err: err}
		}
	}

	h.Set("Content-Type", "application/json")
	_, err = b.Write(scJason)
	if err != nil {
		return weft.StatusError{Code: http.StatusInternalServerError, Err: err}
	}
	return nil
}

func (s *FDSNSensorXML) getSensors(sensorType, station, location string) SensorStation {
	for _, g := range s.Groups {
		tpName := allSensorTypes[sensorType]
		if tpName == g.Name {
			return g.getStation(station, location)
		}
	}
	return nil
}

func (s *Group) getStation(station, location string) SensorStation {
	for _, st := range s.Stations {
		if st.Code == station {
			return st.getSensors4Location(location)
		}
	}
	for _, st := range s.Marks {
		if st.Code == station {
			return st
		}
	}
	for _, st := range s.Mounts {
		if st.Code == station {
			return st.getSensors4Location(location)
		}
	}
	for _, st := range s.Samples {
		if st.Code == station {
			return st.getSensors4Location(location)
		}
	}
	return nil
}

func (s *Station) getSensors4Location(location string) Station {
	result := Station{Code: s.Code,
		Name:      s.Name,
		StartDate: s.StartDate,
		EndDate:   s.EndDate,
		Latitude:  s.Latitude,
		Longitude: s.Longitude}

	for _, st := range s.Sites {
		if location == "" || st.Code == location {
			result.Sites = append(result.Sites, st)
			break
		}
	}
	return result
}

func (s *Mount) getSensors4Location(location string) Mount {
	result := Mount{Code: s.Code,
		Name:      s.Name,
		StartDate: s.StartDate,
		EndDate:   s.EndDate,
		Latitude:  s.Latitude,
		Longitude: s.Longitude}

	for _, st := range s.Views {
		if location == "" || st.Code == location {
			result.Views = append(result.Views, st)
			break
		}
	}
	return result
}

// apply sensor stations as GeoJSON features
func (s *FDSNSensorXML) getStationFeatures(sensorType string, startDate, endDate time.Time) FeatureCollection {
	sTypes := strings.Split(sensorType, ",")
	sensorFeatures := &FeatureCollection{
		Type:     "FeatureCollection",
		Features: []Feature{},
	}
	for _, g := range s.Groups {
		if !checkSensorType(sTypes, g.Name) {
			continue
		}
		for _, st := range g.Stations {
			st.toGeoJSONFeature(g.Name, startDate, endDate, sensorFeatures)
		}
		for _, mk := range g.Marks {
			mk.toGeoJSONFeature(g.Name, startDate, endDate, sensorFeatures)
		}
		for _, sp := range g.Samples {
			sp.toGeoJSONFeature(g.Name, startDate, endDate, sensorFeatures)
		}
		for _, mt := range g.Mounts {
			mt.toGeoJSONFeature(g.Name, startDate, endDate, sensorFeatures)
		}
	}
	return *sensorFeatures
}

func (s *Station) toGeoJSONFeature(sensorType string, startDate, endDate time.Time, fc *FeatureCollection) {
	if s.StartDate.After(endDate) {
		return
	}
	if s.EndDate.Before(startDate) {
		return
	}

	for _, ss := range s.Sites {
		if ss.StartDate.After(endDate) {
			continue
		}
		if ss.EndDate.Before(startDate) {
			continue
		}
		ff := Feature{
			Type: "Feature",
			Properties: featureProperties{
				Code:       s.Code,
				Name:       s.Name,
				Start:      ss.StartDate,
				End:        ss.EndDate,
				SensorType: sensorType,
				Location:   ss.Code,
			},
			Geometry: point{
				Type:        "Point",
				Coordinates: [2]float64{ss.Longitude, ss.Latitude},
			},
		}
		fc.Features = append(fc.Features, ff)

	}
}

func (s *Mark) toGeoJSONFeature(sensorType string, startDate, endDate time.Time, fc *FeatureCollection) {
	if s.StartDate.After(endDate) {
		return
	}
	if s.EndDate.Before(startDate) {
		return
	}

	ff := Feature{
		Type: "Feature",
		Properties: featureProperties{
			Code:       s.Code,
			Name:       s.Name,
			Start:      s.StartDate,
			End:        s.EndDate,
			SensorType: sensorType,
		},
		Geometry: point{
			Type:        "Point",
			Coordinates: [2]float64{s.Longitude, s.Latitude},
		},
	}
	fc.Features = append(fc.Features, ff)
}

func (s *Mount) toGeoJSONFeature(sensorType string, startDate, endDate time.Time, fc *FeatureCollection) {
	if s.StartDate.After(endDate) {
		return
	}
	if s.EndDate.Before(startDate) {
		return
	}

	for _, ss := range s.Views { //TODO sall we do each views? they are the same coordinate
		if ss.StartDate.After(endDate) {
			continue
		}
		if ss.EndDate.Before(startDate) {
			continue
		}
		ff := Feature{
			Type: "Feature",
			Properties: featureProperties{
				Code:       s.Code,
				Name:       s.Name,
				Start:      ss.StartDate,
				End:        ss.EndDate,
				SensorType: sensorType,
				Location:   ss.Code,
			},
			Geometry: point{
				Type:        "Point",
				Coordinates: [2]float64{s.Longitude, s.Latitude},
			},
		}
		fc.Features = append(fc.Features, ff)
	}
}

func checkSensorType(sensorTypes []string, sensorTypeName string) bool {
	for _, tp := range sensorTypes {
		tpName := allSensorTypes[tp]
		if tpName == sensorTypeName {
			return true
		}
	}
	return false
}

// Download station XML from S3
func downloadSensorXML(since time.Time) (by *bytes.Buffer, modified time.Time, err error) {
	s3Client, err := s3.NewWithMaxRetries(100)
	if err != nil {
		return
	}

	log.Println("Check sensor xml file from S3: ", s3SensorBucket+"/"+s3SensorMeta)
	tp, err := s3Client.LastModified(s3SensorBucket, s3SensorMeta, "")
	if err != nil {
		return
	}

	if !tp.After(since) {
		return nil, zeroDateTime, errNotModified
	}

	log.Println("Downloading fdsn sensor xml file from S3: ", s3SensorBucket+"/"+s3SensorMeta)

	by = bytes.NewBuffer(nil)
	err = s3Client.Get(s3SensorBucket, s3SensorMeta, "", by)
	if err != nil {
		return
	}

	modified = tp
	log.Println("Download complete.")
	return
}

func loadSensorXML(by *bytes.Buffer, modified time.Time) (sensorObj fdsnSensorObj, err error) {
	var f FDSNSensorXML
	if err = xml.Unmarshal(by.Bytes(), &f); err != nil {
		return
	}
	// Precondition: There's at least 1 network in the source XML.
	// Else program will crash here.
	log.Printf("Done loading %d or more sensor groups.\n", len(f.Groups))
	sensorObj.modified = modified
	sensorObj.sensors = &f

	return
}

// Periodically update data source
func setupSensorXMLUpdater() {
	s := os.Getenv("STATION_RELOAD_INTERVAL")
	reloadInterval, err := strconv.Atoi(s)
	if err != nil {
		log.Printf("Warning: invalid STATION_RELOAD_INTERVAL env variable, use default value %d instead.\n", DEFAULT_RELOAD_INTERVAL)
		reloadInterval = DEFAULT_RELOAD_INTERVAL
	}
	ticker := time.NewTicker(time.Duration(reloadInterval) * time.Second)
	go func() {
		for range ticker.C {
			by, s3Modified, err := downloadSensorXML(fdsnSensors.modified)
			switch err {
			case errNotModified:
			// Do nothing
			case nil:
				newSensorData, err := loadSensorXML(by, s3Modified)
				if err != nil {
					// errNotModified will be silent
					if err != errNotModified {
						log.Println("Error updating data source:", err)
					}
				} else {
					fdsnSensors.Lock()
					fdsnSensors.sensors = newSensorData.sensors
					fdsnSensors.modified = newSensorData.modified
					fdsnSensors.Unlock()
					log.Println("Data source updated.")
				}
			default:
				log.Println("ERROR: Download XML from S3:", err)
			}
		}
	}()
}
