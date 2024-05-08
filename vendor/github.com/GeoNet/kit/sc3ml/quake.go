package sc3ml

import (
	"fmt"
	"io"
	"time"
)

var alertAge = time.Duration(-60) * time.Minute

// Quake for earthquakes.
type Quake struct {
	PublicID              string
	Type                  string
	AgencyID              string
	ModificationTime      time.Time
	Time                  time.Time
	Latitude              float64
	Longitude             float64
	Depth                 float64
	DepthType             string
	MethodID              string
	EarthModelID          string
	EvaluationMode        string
	EvaluationStatus      string
	UsedPhaseCount        int
	UsedStationCount      int
	StandardError         float64
	AzimuthalGap          float64
	MinimumDistance       float64
	Magnitude             float64
	MagnitudeUncertainty  float64
	MagnitudeType         string
	MagnitudeStationCount int
	Site                  string
}

// Status returns a simplified status.
func (q *Quake) Status() string {
	switch {
	case q.Type == "not existing":
		return "deleted"
	case q.Type == "duplicate":
		return "duplicate"
	case q.EvaluationMode == "manual":
		return "reviewed"
	case q.EvaluationStatus == "confirmed":
		return "reviewed"
	default:
		return "automatic"
	}
}

// Quality returns a simplified quality.
func (q *Quake) Quality() string {
	status := q.Status()

	switch {
	case status == "reviewed":
		return "best"
	case status == "deleted":
		return "deleted"
	case q.UsedPhaseCount >= 20 && q.MagnitudeStationCount >= 10:
		return "good"
	default:
		return "caution"
	}
}

// Alert returns true if the quake should be considered for alerting, false
// with a reason if not.
func (q *Quake) Alert() (bool, string) {
	switch {
	case q.Status() == "deleted":
		return false, fmt.Sprintf("%s status deleted not suitable for alerting.", q.PublicID)
	case q.Status() == "duplicate":
		return false, fmt.Sprintf("%s status duplicate not suitable for alerting.", q.PublicID)
	case q.Status() == "automatic" && (q.UsedPhaseCount < 20 || q.MagnitudeStationCount < 10):
		return false, fmt.Sprintf("%s unreviewed with %d phases and %d magnitudes not suitable for alerting.", q.PublicID, q.UsedPhaseCount, q.MagnitudeStationCount)
	case q.Status() == "automatic" && !(q.Depth >= 0.1 && q.AzimuthalGap <= 320.0 && q.MinimumDistance <= 2.5):
		return false, fmt.Sprintf("%s automatic with poor location criteria", q.PublicID)
	case q.Time.Before(time.Now().UTC().Add(alertAge)):
		return false, fmt.Sprintf("%s to old for alerting", q.PublicID)
	default:
		return true, ""
	}
}

// manual returns true if the quake has been manually reviewed in some way.
func (q *Quake) Manual() bool {
	switch {
	case q.Type == "not existing":
		return true
	case q.Type == "duplicate":
		return true
	case q.EvaluationMode == "manual":
		return true
	case q.EvaluationStatus == "confirmed":
		return true
	default:
		return false
	}
}

func FromSC3ML(r io.Reader) (Quake, error) {
	var s Seiscomp

	b, err := io.ReadAll(r)
	if err != nil {
		return Quake{}, err
	}

	err = Unmarshal(b, &s)
	if err != nil {
		return Quake{}, err
	}

	return Quake{
		PublicID:              s.EventParameters.Events[0].PublicID,
		Type:                  s.EventParameters.Events[0].Type,
		AgencyID:              s.EventParameters.Events[0].CreationInfo.AgencyID,
		MethodID:              s.EventParameters.Events[0].PreferredOrigin.MethodID,
		EarthModelID:          s.EventParameters.Events[0].PreferredOrigin.EarthModelID,
		EvaluationMode:        s.EventParameters.Events[0].PreferredOrigin.EvaluationMode,
		EvaluationStatus:      s.EventParameters.Events[0].PreferredOrigin.EvaluationStatus,
		DepthType:             s.EventParameters.Events[0].PreferredOrigin.DepthType,
		MagnitudeType:         s.EventParameters.Events[0].PreferredMagnitude.Type,
		Time:                  s.EventParameters.Events[0].PreferredOrigin.Time.Value,
		Latitude:              s.EventParameters.Events[0].PreferredOrigin.Latitude.Value,
		Longitude:             s.EventParameters.Events[0].PreferredOrigin.Longitude.Value,
		Depth:                 s.EventParameters.Events[0].PreferredOrigin.Depth.Value,
		StandardError:         s.EventParameters.Events[0].PreferredOrigin.Quality.StandardError,
		AzimuthalGap:          s.EventParameters.Events[0].PreferredOrigin.Quality.AzimuthalGap,
		MinimumDistance:       s.EventParameters.Events[0].PreferredOrigin.Quality.MinimumDistance,
		UsedPhaseCount:        int(s.EventParameters.Events[0].PreferredOrigin.Quality.UsedPhaseCount),
		UsedStationCount:      int(s.EventParameters.Events[0].PreferredOrigin.Quality.UsedStationCount),
		MagnitudeStationCount: int(s.EventParameters.Events[0].PreferredMagnitude.StationCount),
		Magnitude:             s.EventParameters.Events[0].PreferredMagnitude.Magnitude.Value,
		MagnitudeUncertainty:  s.EventParameters.Events[0].PreferredMagnitude.Magnitude.Uncertainty,
		ModificationTime:      s.EventParameters.Events[0].ModificationTime,
	}, nil
}

// Publish returns true if the quake should be considered for publishing.
func (q *Quake) Publish() bool {
	switch q.Site {
	case "primary":
		return true
	case "backup":
		return q.Manual()
	default:
		return false
	}
}

// Certainty returns the CAP certainty for the quake.
func (q *Quake) Certainty() string {
	status := q.Status()

	switch {
	case status == `reviewed`:
		return `Observed`
	case status == `deleted`:
		return `Unlikely`
	case q.UsedPhaseCount >= 20 && q.MagnitudeStationCount >= 10:
		return `Likely`
	case status == "automatic" && (q.UsedPhaseCount < 20 || q.MagnitudeStationCount < 10):
		return `Possible`
	default:
		return `Unknown`
	}
}
