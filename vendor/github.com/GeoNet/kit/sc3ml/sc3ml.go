/*
Package sc3ml is for parsing SeisComPML.
*/
package sc3ml

import (
	"bytes"
	"encoding/xml"
	"errors"
	"time"
)

const (
	sc3ml07 = `http://geofon.gfz-potsdam.de/ns/seiscomp3-schema/0.7`
	sc3ml08 = `http://geofon.gfz-potsdam.de/ns/seiscomp3-schema/0.8`
	sc3ml09 = `http://geofon.gfz-potsdam.de/ns/seiscomp3-schema/0.9`
	sc3ml10 = `http://geofon.gfz-potsdam.de/ns/seiscomp3-schema/0.10`
	sc3ml11 = `http://geofon.gfz-potsdam.de/ns/seiscomp3-schema/0.11`
)

type Seiscomp struct {
	XMLns           string          `xml:"xmlns,attr"`
	EventParameters EventParameters `xml:"EventParameters"`
}

type EventParameters struct {
	Events     []Event     `xml:"event"`
	Picks      []Pick      `xml:"pick"`
	Amplitudes []Amplitude `xml:"amplitude"`
	Origins    []Origin    `xml:"origin"`
}

type Event struct {
	PublicID             string `xml:"publicID,attr"`
	PreferredOriginID    string `xml:"preferredOriginID"`
	PreferredMagnitudeID string `xml:"preferredMagnitudeID"`
	Type                 string `xml:"type"`
	PreferredOrigin      Origin
	PreferredMagnitude   Magnitude
	ModificationTime     time.Time    `xml:"-"` // most recent modification time for all objects in the event.  Not in the XML.
	CreationInfo         CreationInfo `xml:"creationInfo"`
}

type CreationInfo struct {
	AgencyID         string    `xml:"agencyID"`
	CreationTime     time.Time `xml:"creationTime"`
	ModificationTime time.Time `xml:"modificationTime"`
}

type Origin struct {
	PublicID          string             `xml:"publicID,attr"`
	Time              TimeValue          `xml:"time"`
	Latitude          RealQuantity       `xml:"latitude"`
	Longitude         RealQuantity       `xml:"longitude"`
	Depth             RealQuantity       `xml:"depth"`
	DepthType         string             `xml:"depthType"`
	MethodID          string             `xml:"methodID"`
	EarthModelID      string             `xml:"earthModelID"`
	Quality           Quality            `xml:"quality"`
	EvaluationMode    string             `xml:"evaluationMode"`
	EvaluationStatus  string             `xml:"evaluationStatus"`
	Arrivals          []Arrival          `xml:"arrival"`
	StationMagnitudes []StationMagnitude `xml:"stationMagnitude"`
	Magnitudes        []Magnitude        `xml:"magnitude"`
}

type Quality struct {
	UsedPhaseCount   int64   `xml:"usedPhaseCount"`
	UsedStationCount int64   `xml:"usedStationCount"`
	StandardError    float64 `xml:"standardError"`
	AzimuthalGap     float64 `xml:"azimuthalGap"`
	MinimumDistance  float64 `xml:"minimumDistance"`
}

type Arrival struct {
	PickID       string  `xml:"pickID"`
	Phase        string  `xml:"phase"`
	Azimuth      float64 `xml:"azimuth"`
	Distance     float64 `xml:"distance"`
	TimeResidual float64 `xml:"timeResidual"`
	Weight       float64 `xml:"weight"`
	Pick         Pick
}

type Pick struct {
	PublicID         string     `xml:"publicID,attr"`
	Time             TimeValue  `xml:"time"`
	WaveformID       WaveformID `xml:"waveformID"`
	EvaluationMode   string     `xml:"evaluationMode"`
	EvaluationStatus string     `xml:"evaluationStatus"`
}

type WaveformID struct {
	NetworkCode  string `xml:"networkCode,attr"`
	StationCode  string `xml:"stationCode,attr"`
	LocationCode string `xml:"locationCode,attr"`
	ChannelCode  string `xml:"channelCode,attr"`
}

type RealQuantity struct {
	Value       float64 `xml:"value"`
	Uncertainty float64 `xml:"uncertainty"`
}

type TimeValue struct {
	Value time.Time `xml:"value"`
}

type Magnitude struct {
	PublicID                      string                         `xml:"publicID,attr"`
	Magnitude                     RealQuantity                   `xml:"magnitude"`
	Type                          string                         `xml:"type"`
	MethodID                      string                         `xml:"methodID"`
	StationCount                  int64                          `xml:"stationCount"`
	StationMagnitudeContributions []StationMagnitudeContribution `xml:"stationMagnitudeContribution"`
}

type StationMagnitudeContribution struct {
	StationMagnitudeID string  `xml:"stationMagnitudeID"`
	Weight             float64 `xml:"weight"`
	Residual           float64 `xml:"residual"`
	StationMagnitude   StationMagnitude
}

type StationMagnitude struct {
	PublicID    string       `xml:"publicID,attr"`
	Magnitude   RealQuantity `xml:"magnitude"`
	Type        string       `xml:"type"`
	AmplitudeID string       `xml:"amplitudeID"`
	WaveformID  WaveformID   `xml:"waveformID"`
	Amplitude   Amplitude
}

type Amplitude struct {
	PublicID  string       `xml:"publicID,attr"`
	Amplitude RealQuantity `xml:"amplitude"`
	PickID    string       `xml:"pickID"`
	Azimuth   float64      // not in the SC3ML - will be mapped from arrival using PickID
	Distance  float64      // not in the SC3ML - will be mapped from arrival using PickID
}

// Unmarshal unmarshals the SeisComPML in b and initialises all
// the objects referenced by ID in the SeisComPML e.g., PreferredOrigin,
// PreferredMagnitude etc.
//
// Supported SC3ML versions are 0.7, 0.8, 0.9, 0.10, 0.11
// Any other versions will result in a error.
func Unmarshal(b []byte, s *Seiscomp) error {
	if err := xml.Unmarshal(b, s); err != nil {
		return err
	}

	switch s.XMLns {
	case sc3ml07:
	case sc3ml08:
	case sc3ml09:
	case sc3ml10:
	case sc3ml11:
	default:
		return errors.New("unsupported SC3ML version")
	}

	var picks = make(map[string]Pick)
	for k, v := range s.EventParameters.Picks {
		picks[v.PublicID] = s.EventParameters.Picks[k]
	}

	var arrivals = make(map[string]Arrival)
	for i := range s.EventParameters.Origins {
		for _, v := range s.EventParameters.Origins[i].Arrivals {
			arrivals[v.PickID] = v
		}
	}

	var amplitudes = make(map[string]Amplitude)
	for k, v := range s.EventParameters.Amplitudes {
		a := s.EventParameters.Amplitudes[k]

		// add distance and azimuth from the arrival with the matching PickID.
		pk := arrivals[v.PickID]

		a.Distance = pk.Distance
		a.Azimuth = pk.Azimuth

		amplitudes[v.PublicID] = a
	}

	for i := range s.EventParameters.Origins {
		for k, v := range s.EventParameters.Origins[i].Arrivals {
			s.EventParameters.Origins[i].Arrivals[k].Pick = picks[v.PickID]
		}

		var stationMagnitudes = make(map[string]StationMagnitude)

		for k, v := range s.EventParameters.Origins[i].StationMagnitudes {
			s.EventParameters.Origins[i].StationMagnitudes[k].Amplitude = amplitudes[v.AmplitudeID]
			stationMagnitudes[v.PublicID] = s.EventParameters.Origins[i].StationMagnitudes[k]
		}

		for j := range s.EventParameters.Origins[i].Magnitudes {
			for k, v := range s.EventParameters.Origins[i].Magnitudes[j].StationMagnitudeContributions {
				s.EventParameters.Origins[i].Magnitudes[j].StationMagnitudeContributions[k].StationMagnitude = stationMagnitudes[v.StationMagnitudeID]
			}
		}
	}

	// set the preferred origin.
	// set the preferred mag which can come from any origin
	for i := range s.EventParameters.Events {
		for k, v := range s.EventParameters.Origins {
			if v.PublicID == s.EventParameters.Events[i].PreferredOriginID {
				s.EventParameters.Events[i].PreferredOrigin = s.EventParameters.Origins[k]
			}
			for _, mag := range v.Magnitudes {
				if mag.PublicID == s.EventParameters.Events[i].PreferredMagnitudeID {
					s.EventParameters.Events[i].PreferredMagnitude = mag
				}
			}
		}
	}

	// set the most recent modified time as long as there is only one event.
	// XML token parse the entire SC3ML, looking for creationInfo.
	// assumes all objects in the SC3ML are associated, or have been associated,
	// with the event somehow.
	if len(s.EventParameters.Events) != 1 {
		return nil
	}

	var by bytes.Buffer
	by.Write(b)
	d := xml.NewDecoder(&by)
	var tk xml.Token
	var err error

	for {
		// Read tokens from the XML document in a stream.
		tk, err = d.Token()
		if tk == nil {
			break
		}
		if err != nil {
			return err
		}

		switch se := tk.(type) {
		case xml.StartElement:
			if se.Name.Local == "creationInfo" {
				var c CreationInfo
				err = d.DecodeElement(&c, &se)
				if err != nil {
					return err
				}
				if c.ModificationTime.After(s.EventParameters.Events[0].ModificationTime) {
					s.EventParameters.Events[0].ModificationTime = c.ModificationTime
				}
				if c.CreationTime.After(s.EventParameters.Events[0].ModificationTime) {
					s.EventParameters.Events[0].ModificationTime = c.CreationTime
				}
			}
		}
	}

	return nil
}
