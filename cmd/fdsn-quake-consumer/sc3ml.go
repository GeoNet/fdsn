package main

import (
	"bytes"
	"fmt"
	"github.com/GeoNet/kit/sc3ml"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	"io"
	"log"
	"os/exec"
	"time"
)

const deleted = `not existing`

var sc3ml07 = []byte(`<seiscomp xmlns="http://geofon.gfz-potsdam.de/ns/seiscomp3-schema/0.7" version="0.7">`)
var sc3ml08 = []byte(`<seiscomp xmlns="http://geofon.gfz-potsdam.de/ns/seiscomp3-schema/0.8" version="0.8">`)
var sc3ml09 = []byte(`<seiscomp xmlns="http://geofon.gfz-potsdam.de/ns/seiscomp3-schema/0.9" version="0.9">`)
var sc3ml10 = []byte(`<seiscomp xmlns="http://geofon.gfz-potsdam.de/ns/seiscomp3-schema/0.10" version="0.10">`)

// event is for saving information to the db.
// field names must match the column names in fdsn.event and the field names must be exported.
// The origin_geom column is added with a DB trigger.
// Capitalization of database fdsn.event column names is not significant.
type event struct {
	PublicID              string
	EventType             string
	ModificationTime      time.Time
	OriginTime            time.Time
	Longitude             float64
	Latitude              float64
	Depth                 float64
	DepthType             string
	EvaluationMethod      string
	EarthModel            string
	EvaluationMode        string
	EvaluationStatus      string
	UsedPhaseCount        int64
	UsedStationCount      int64
	OriginError           float64
	AzimuthalGap          float64
	MinimumDistance       float64
	Magnitude             float64
	MagnitudeUncertainty  float64
	MagnitudeType         string
	MagnitudeStationCount int64
	Deleted               bool
	Sc3ml                 string // complete SeisComPML - any version.
	Quakeml12Event        string // a QuakeML 1.2 event fragment.
}

/*
toQuakeMLEvent converts seisComPML to a QuakeML event fragment using an XSLT.
Supported versions of SC3ML are

   * 0.7
   * 0.8
   * 0.9
   * 0.10

The xslt source is from http://geofon.gfz-potsdam.de/ns/seiscomp3-schema/0.7/sc3ml_0.7__quakeml_1.2.xsl
It has been edited to output only an Event fragment without the parent elements and namespaces.  e.g.,

    129d128
    <         xmlns="http://quakeml.org/xmlns/bed/1.2"
    132c131
    <     <xsl:output method="xml" encoding="UTF-8" indent="yes"/>
    ---
    >     <xsl:output method="xml" encoding="UTF-8" indent="no" omit-xml-declaration="yes"/>
    138c137
    <     <xsl:variable name="ID_PREFIX" select="'smi:org.gfz-potsdam.de/geofon/'"/>
    ---
    >     <xsl:variable name="ID_PREFIX" select="'smi:nz.org.geonet/'"/>
    145d143
    <         <q:quakeml>
    147d144
    <                 <eventParameters>
    156d152
    <                 </eventParameters>
    158d153
    <         </q:quakeml>

*/
func toQuakeMLEvent(seisComPML []byte) (string, error) {
	cmd := exec.Command("/usr/bin/xsltproc")

	switch {
	case bytes.Contains(seisComPML, sc3ml07):
		cmd.Args = append(cmd.Args, "assets/sc3ml_0.7__quakeml_1.2.xsl")
	case bytes.Contains(seisComPML, sc3ml08):
		cmd.Args = append(cmd.Args, "assets/sc3ml_0.8__quakeml_1.2.xsl")
	case bytes.Contains(seisComPML, sc3ml09):
		cmd.Args = append(cmd.Args, "assets/sc3ml_0.9__quakeml_1.2.xsl")
	case bytes.Contains(seisComPML, sc3ml10):
		cmd.Args = append(cmd.Args, "assets/sc3ml_0.10__quakeml_1.2.xsl")

	default:
		return "", fmt.Errorf("found no %s", "XSLT")
	}

	cmd.Args = append(cmd.Args, "-")

	var err error
	var in io.WriteCloser
	var out io.ReadCloser
	var b bytes.Buffer

	if in, err = cmd.StdinPipe(); err != nil {
		return "", err
	}
	defer in.Close()

	if out, err = cmd.StdoutPipe(); err != nil {
		return "", err
	}
	defer out.Close()

	if err := cmd.Start(); err != nil {
		return "", err
	}
	if _, err = in.Write(seisComPML); err != nil {
		return "", err
	}

	if err = in.Close(); err != nil {
		return "", err
	}

	if _, err = b.ReadFrom(out); err != nil {
		return "", err
	}

	err = cmd.Wait()

	return b.String(), err
}

// unmarshal unmarshals seisComPML into event.
func unmarshal(seisComPML []byte, e *event) error {
	var s sc3ml.Seiscomp
	var err error

	if err = sc3ml.Unmarshal(seisComPML, &s); err != nil {
		return errors.Wrapf(err, "unmarshaling SC3ML")
	}

	if len(s.EventParameters.Events) != 1 {
		return errors.Errorf("expected 1 event, got %d", len(s.EventParameters.Events))
	}

	e.PublicID = s.EventParameters.Events[0].PublicID
	e.EventType = s.EventParameters.Events[0].Type
	e.ModificationTime = s.EventParameters.Events[0].ModificationTime
	e.OriginTime = s.EventParameters.Events[0].PreferredOrigin.Time.Value
	e.Longitude = s.EventParameters.Events[0].PreferredOrigin.Longitude.Value
	e.Latitude = s.EventParameters.Events[0].PreferredOrigin.Latitude.Value
	e.Depth = s.EventParameters.Events[0].PreferredOrigin.Depth.Value
	e.DepthType = s.EventParameters.Events[0].PreferredOrigin.DepthType
	e.EvaluationMethod = s.EventParameters.Events[0].PreferredOrigin.MethodID
	e.EarthModel = s.EventParameters.Events[0].PreferredOrigin.EarthModelID
	e.EvaluationMode = s.EventParameters.Events[0].PreferredOrigin.EvaluationMode
	e.EvaluationStatus = s.EventParameters.Events[0].PreferredOrigin.EvaluationStatus
	e.UsedPhaseCount = s.EventParameters.Events[0].PreferredOrigin.Quality.UsedPhaseCount
	e.UsedStationCount = s.EventParameters.Events[0].PreferredOrigin.Quality.UsedStationCount
	e.OriginError = s.EventParameters.Events[0].PreferredOrigin.Quality.StandardError
	e.AzimuthalGap = s.EventParameters.Events[0].PreferredOrigin.Quality.AzimuthalGap
	e.MinimumDistance = s.EventParameters.Events[0].PreferredOrigin.Quality.MinimumDistance
	e.Magnitude = s.EventParameters.Events[0].PreferredMagnitude.Magnitude.Value
	e.MagnitudeUncertainty = s.EventParameters.Events[0].PreferredMagnitude.Magnitude.Uncertainty
	e.MagnitudeType = s.EventParameters.Events[0].PreferredMagnitude.Type
	e.MagnitudeStationCount = s.EventParameters.Events[0].PreferredMagnitude.StationCount
	e.Deleted = s.EventParameters.Events[0].Type == deleted
	e.Sc3ml = string(seisComPML)

	if e.Quakeml12Event, err = toQuakeMLEvent(seisComPML); err != nil {
		return fmt.Errorf("XSLT transform %s: %s", s.EventParameters.Events[0].PublicID, err.Error())
	}

	return nil
}

// save or update event information in the DB to be the latest (most recent) information.
func (e *event) save() error {
	// convert e to a map[string]interface{} and use that to build the DB insert statement.
	txn, err := db.Begin()
	if err != nil {
		return err
	}

	_, err = txn.Exec(`DELETE FROM fdsn.event WHERE PublicID = $1 AND ModificationTime <= $2`, e.PublicID, e.ModificationTime)
	if err != nil {
		if e := txn.Rollback(); e != nil {
			log.Printf("Rollback Failed: %v", e)
		}
		return err
	}

	_, err = txn.Exec(`INSERT INTO fdsn.event(PublicID, EventType, ModificationTime, OriginTime, Longitude, Latitude,
			Depth, DepthType, EvaluationMethod, EarthModel, EvaluationMode, EvaluationStatus, UsedPhaseCount,
			UsedStationCount, OriginError, AzimuthalGap, MinimumDistance, Magnitude, MagnitudeUncertainty, MagnitudeType,
			MagnitudeStationCount, Deleted, Sc3ml, Quakeml12Event) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11,
			$12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24)`,
		e.PublicID, e.EventType, e.ModificationTime, e.OriginTime, e.Longitude, e.Latitude, e.Depth, e.DepthType,
		e.EvaluationMethod, e.EarthModel, e.EvaluationMode, e.EvaluationStatus, e.UsedPhaseCount, e.UsedStationCount,
		e.OriginError, e.AzimuthalGap, e.MinimumDistance, e.Magnitude, e.MagnitudeUncertainty, e.MagnitudeType,
		e.MagnitudeStationCount, e.Deleted, e.Sc3ml, e.Quakeml12Event)
	switch err {
	case nil:
		err = txn.Commit()
		if err != nil {
			return err
		}
		return nil
	default:
		// a unique_violation means the new event info is older than in the table already.
		// this is not an error for this application - we want the latest information only in
		// the event table.
		// http://www.postgresql.org/docs/9.3/static/errcodes-appendix.html
		if errorUnique, ok := err.(*pq.Error); ok {
			if errorUnique.Code == `23505` {
				err = nil
			}
			if e := txn.Rollback(); e != nil {
				log.Printf("Rollback Failed: %v", e)
			}
			return err
		} else {
			// non-pq error
			if e := txn.Rollback(); e != nil {
				log.Printf("Rollback Failed: %v", e)
			}
			return err
		}
	}
}
