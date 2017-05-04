package msg

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"sort"
	"strconv"
	"time"
)

type sc3ml07 struct {
	EventParameters eventParameters `xml:"EventParameters"`
	Error           error
}

type eventParameters struct {
	Event   event    `xml:"event"`
	Origins []origin `xml:"origin"`
}

type event struct {
	PublicID             string `xml:"publicID,attr"`
	PreferredOriginID    string `xml:"preferredOriginID"`
	PreferredMagnitudeID string `xml:"preferredMagnitudeID"`
	Type                 string `xml:"type"`
	PreferredOrigin      origin
	PreferredMagnitude   magnitude
	CreationInfo         creationInfo `xml:"creationInfo"`
}

type creationInfo struct {
	AgencyID         string `xml:"agencyID"`
	ModificationTime string `xml:"modificationTime"`
	CreationTime     string `xml:"creationTime"`
}

type origin struct {
	PublicID         string       `xml:"publicID,attr"`
	Time             timeValue    `xml:"time"`
	Magnitudes       []magnitude  `xml:"magnitude"`
	Latitude         value        `xml:"latitude"`
	Longitude        value        `xml:"longitude"`
	Depth            value        `xml:"depth"`
	DepthType        string       `xml:"depthType"`
	Quality          quality      `xml:"quality"`
	MethodID         string       `xml:"methodID"`
	EarthModelID     string       `xml:"earthModelID"`
	EvaluationMode   string       `xml:"evaluationMode"`
	EvaluationStatus string       `xml:"evaluationStatus"`
	CreationInfo     creationInfo `xml:"creationInfo"`
}

type quality struct {
	UsedPhaseCount   string `xml:"usedPhaseCount"`
	UsedStationCount string `xml:"usedStationCount"`
	StandardError    string `xml:"standardError"`
	AzimuthalGap     string `xml:"azimuthalGap"`
	MinimumDistance  string `xml:"minimumDistance"`
}

type value struct {
	Value       string `xml:"value"`
	Uncertainty string `xml:"uncertainty"`
}

type timeValue struct {
	Value       string `xml:"value"`
	Uncertainty string `xml:"uncertainty"`
}

type magnitudeValue struct {
	Value       string `xml:"value"`
	Uncertainty string `xml:"uncertainty"`
}

type magnitude struct {
	PublicID       string         `xml:"publicID,attr"`
	MagnitudeValue magnitudeValue `xml:"magnitude"`
	Type           string         `xml:"type"`
	MethodID       string         `xml:"methodID"`
	StationCount   string         `xml:"stationCount"`
	CreationInfo   creationInfo   `xml:"creationInfo"`
}

// init performs initialisation functions on the SeisCompML.  Should be called called after unmarshal.
func (s *sc3ml07) init() {
	if s.Error != nil {
		return
	}

	if s.EventParameters.Event.PreferredOriginID == "" {
		s.Error = fmt.Errorf("empty PreferredOriginID")
		return
	}

	if s.EventParameters.Event.PreferredMagnitudeID == "" {
		s.Error = fmt.Errorf("empty PreferredMagnitudeID")
		return
	}

	magFound := false

	for _, origin := range s.EventParameters.Origins {
		for _, mag := range origin.Magnitudes {
			if mag.PublicID == s.EventParameters.Event.PreferredMagnitudeID {
				s.EventParameters.Event.PreferredMagnitude = mag
				magFound = true
			}
		}
	}

	if !magFound {
		s.Error = fmt.Errorf("found no magnitude matching PreferredMagnitudeID %s", s.EventParameters.Event.PreferredMagnitudeID)
		return
	}

	originFound := false

	for _, origin := range s.EventParameters.Origins {
		if origin.PublicID == s.EventParameters.Event.PreferredOriginID {
			s.EventParameters.Event.PreferredOrigin = origin
			originFound = true
		}
	}

	if !originFound {
		s.Error = fmt.Errorf("found no origin matching PreferredOriginID %s", s.EventParameters.Event.PreferredOriginID)
		return
	}

	return
}

func ReadSC3ML07(filename string) Quake {
	s := sc3ml07{}

	b, err := ioutil.ReadFile(filename)
	if err != nil {
		s.Error = err
		return s.quake()
	}

	s.Error = xml.Unmarshal(b, &s)
	s.init()
	return s.quake()
}

// returns the most recent modificationTime or creationTime for the
// components of the SC3ML that we are interested in for a Quake.
// Returns without processing if s.Error is not nil
func (s *sc3ml07) quakeModTime() (time.Time, error) {
	if s.Error != nil {
		return time.Time{}, s.Error
	}

	var t []string

	if s.EventParameters.Event.CreationInfo.CreationTime != "" {
		t = append(t, s.EventParameters.Event.CreationInfo.CreationTime)
	}

	if s.EventParameters.Event.CreationInfo.ModificationTime != "" {
		t = append(t, s.EventParameters.Event.CreationInfo.ModificationTime)
	}

	if s.EventParameters.Event.PreferredOrigin.CreationInfo.CreationTime != "" {
		t = append(t, s.EventParameters.Event.PreferredOrigin.CreationInfo.CreationTime)
	}

	if s.EventParameters.Event.PreferredOrigin.CreationInfo.ModificationTime != "" {
		t = append(t, s.EventParameters.Event.PreferredOrigin.CreationInfo.ModificationTime)
	}

	if s.EventParameters.Event.PreferredMagnitude.CreationInfo.CreationTime != "" {
		t = append(t, s.EventParameters.Event.PreferredMagnitude.CreationInfo.CreationTime)
	}

	if s.EventParameters.Event.PreferredMagnitude.CreationInfo.ModificationTime != "" {
		t = append(t, s.EventParameters.Event.PreferredMagnitude.CreationInfo.ModificationTime)
	}

	if !(len(t) >= 1) {
		return time.Time{}, fmt.Errorf("found no candidates for setting modificationTime for  %s", s.EventParameters.Event.PublicID)
	}

	sort.Sort(sort.Reverse(sort.StringSlice(t)))

	return time.Parse(time.RFC3339Nano, t[0])
}

// quake is safe to use with self closing or missing XML tags.
// Returns q with q.Error set if s.Error is not nil.
func (s *sc3ml07) quake() (q Quake) {
	if s.Error != nil {
		q.err = fmt.Errorf("quake created from errored SC3ML07: %s", s.Error.Error())
		return
	}

	q.PublicID = s.EventParameters.Event.PublicID
	q.Type = s.EventParameters.Event.Type
	q.AgencyID = s.EventParameters.Event.CreationInfo.AgencyID
	q.MethodID = s.EventParameters.Event.PreferredOrigin.MethodID
	q.EarthModelID = s.EventParameters.Event.PreferredOrigin.EarthModelID
	q.EvaluationMode = s.EventParameters.Event.PreferredOrigin.EvaluationMode
	q.EvaluationStatus = s.EventParameters.Event.PreferredOrigin.EvaluationStatus
	q.DepthType = s.EventParameters.Event.PreferredOrigin.DepthType
	q.MagnitudeType = s.EventParameters.Event.PreferredMagnitude.Type

	mt, err := s.quakeModTime()
	if err != nil {
		q.err = fmt.Errorf("setting modificationTime: %s", err.Error())
		return
	} else {
		q.ModificationTime = mt
	}

	if s.EventParameters.Event.PreferredOrigin.Time.Value != "" {
		tm, err := time.Parse(time.RFC3339Nano, s.EventParameters.Event.PreferredOrigin.Time.Value)
		if err != nil {
			q.err = fmt.Errorf("parsing s.EventParameters.Event.PreferredOrigin.Time.Value: %s", err.Error())
			return
		} else {
			q.Time = tm
		}
	}

	if s.EventParameters.Event.PreferredOrigin.Latitude.Value != "" {
		n, err := strconv.ParseFloat(s.EventParameters.Event.PreferredOrigin.Latitude.Value, 64)
		if err != nil {
			q.err = fmt.Errorf("parsing s.EventParameters.Event.PreferredOrigin.Latitude.Value: %s", err.Error())
			return
		} else {
			q.Latitude = n
		}
	}

	if s.EventParameters.Event.PreferredOrigin.Longitude.Value != "" {
		n, err := strconv.ParseFloat(s.EventParameters.Event.PreferredOrigin.Longitude.Value, 64)
		if err != nil {
			q.err = fmt.Errorf("parsing s.EventParameters.Event.PreferredOrigin.Longitude.Value: %s", err.Error())
			return
		} else {
			q.Longitude = n
		}
	}

	if s.EventParameters.Event.PreferredOrigin.Depth.Value != "" {
		n, err := strconv.ParseFloat(s.EventParameters.Event.PreferredOrigin.Depth.Value, 64)
		if err != nil {
			q.err = fmt.Errorf("parsing s.EventParameters.Event.PreferredOrigin.Longitude.Value: %s", err.Error())
			return
		} else {
			q.Depth = n
		}
	}

	if s.EventParameters.Event.PreferredOrigin.Quality.StandardError != "" {
		n, err := strconv.ParseFloat(s.EventParameters.Event.PreferredOrigin.Quality.StandardError, 64)
		if err != nil {
			q.err = fmt.Errorf("parsing s.EventParameters.Event.PreferredOrigin.Quality.StandardError: %s", err.Error())
			return
		} else {
			q.StandardError = n
		}
	}

	if s.EventParameters.Event.PreferredOrigin.Quality.AzimuthalGap != "" {
		n, err := strconv.ParseFloat(s.EventParameters.Event.PreferredOrigin.Quality.AzimuthalGap, 64)
		if err != nil {
			q.err = fmt.Errorf("parsing s.EventParameters.Event.PreferredOrigin.Quality.AzimuthalGap: %s", err.Error())
			return
		} else {
			q.AzimuthalGap = n
		}
	}

	if s.EventParameters.Event.PreferredOrigin.Quality.MinimumDistance != "" {
		n, err := strconv.ParseFloat(s.EventParameters.Event.PreferredOrigin.Quality.MinimumDistance, 64)
		if err != nil {
			q.err = fmt.Errorf("parsing s.EventParameters.Event.PreferredOrigin.Quality.MinimumDistance: %s", err.Error())
			return
		} else {
			q.MinimumDistance = n
		}
	}

	if s.EventParameters.Event.PreferredOrigin.Quality.UsedPhaseCount != "" {
		i, err := strconv.Atoi(s.EventParameters.Event.PreferredOrigin.Quality.UsedPhaseCount)
		if err != nil {
			q.err = fmt.Errorf("parsing s.EventParameters.Event.PreferredOrigin.Quality.UsedPhaseCount: %s", err.Error())
			return
		} else {
			q.UsedPhaseCount = i
		}
	}

	if s.EventParameters.Event.PreferredOrigin.Quality.UsedStationCount != "" {
		i, err := strconv.Atoi(s.EventParameters.Event.PreferredOrigin.Quality.UsedStationCount)
		if err != nil {
			q.err = fmt.Errorf("parsing s.EventParameters.Event.PreferredOrigin.Quality.UsedPhaseCount: %s", err.Error())
			return
		} else {
			q.UsedStationCount = i
		}
	}

	if s.EventParameters.Event.PreferredMagnitude.StationCount != "" {
		i, err := strconv.Atoi(s.EventParameters.Event.PreferredMagnitude.StationCount)
		if err != nil {
			q.err = fmt.Errorf("parsing s.EventParameters.Event.PreferredMagnitude.StationCount: %s", err.Error())
			return
		} else {
			q.MagnitudeStationCount = i
		}
	}

	if s.EventParameters.Event.PreferredMagnitude.MagnitudeValue.Value != "" {
		n, err := strconv.ParseFloat(s.EventParameters.Event.PreferredMagnitude.MagnitudeValue.Value, 64)
		if err != nil {
			q.err = fmt.Errorf("parsing s.EventParameters.Event.PreferredMagnitude.MagnitudeValue.Value: %s", err.Error())
			return
		} else {
			q.Magnitude = n
		}
	}

	if s.EventParameters.Event.PreferredMagnitude.MagnitudeValue.Uncertainty != "" {
		n, err := strconv.ParseFloat(s.EventParameters.Event.PreferredMagnitude.MagnitudeValue.Uncertainty, 64)
		if err != nil {
			q.err = fmt.Errorf("parsing s.EventParameters.Event.PreferredMagnitude.MagnitudeValue.Uncertainty: %s", err.Error())
			return
		} else {
			q.MagnitudeUncertainty = n
		}
	}

	return
}
