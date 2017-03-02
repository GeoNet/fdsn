package main

import (
	"bytes"
	"time"
)

type AngleType struct {
	Value      float64 `xml:",chardata"`
	Unit       string  `xml:"unit,attr"`
	PlusError  float64 `xml:"plusError,attr"`
	MinusError float64 `xml:"minusError,attr"`
}

// May be one of MACLAURIN
type ApproximationType string

// Instrument azimuth, degrees clockwise from North.
type AzimuthType struct {
	Value      float64 `xml:",chardata"`
	Unit       string  `xml:"unit,attr"`
	PlusError  float64 `xml:"plusError,attr"`
	MinusError float64 `xml:"minusError,attr"`
}

type BaseFilterType struct {
	ResourceId  string    `xml:"resourceId,attr"`
	Name        string    `xml:"name,attr"`
	Items       []string  `xml:",any"`
	Description string    `xml:"Description"`
	InputUnits  UnitsType `xml:"InputUnits"`
	OutputUnits UnitsType `xml:"OutputUnits"`
}

type BaseNodeType struct {
	Code             string               `xml:"code,attr"`
	StartDate        xsdDateTime          `xml:"startDate,attr"`
	EndDate          xsdDateTime          `xml:"endDate,attr"`
	RestrictedStatus RestrictedStatusType `xml:"restrictedStatus,attr"`
	AlternateCode    string               `xml:"alternateCode,attr"`
	HistoricalCode   string               `xml:"historicalCode,attr"`
	Items            []string             `xml:",any"`
	Description      string               `xml:"Description"`
	Comment          []CommentType        `xml:"Comment"`
}

// May be one of ANALOG (RADIANS/SECOND), ANALOG (HERTZ), DIGITAL
type CfTransferFunctionType string

// Equivalent to SEED blockette 52 and parent element for the related the
// response blockettes.
type ChannelType struct {
	BaseNodeType
	Unit              string                  `xml:"unit,attr"`
	LocationCode      string                  `xml:"locationCode,attr"`
	ExternalReference []ExternalReferenceType `xml:"ExternalReference"`
	Latitude          LatitudeType            `xml:"Latitude"`
	Longitude         LongitudeType           `xml:"Longitude"`
	Elevation         DistanceType            `xml:"Elevation"`
	Depth             DistanceType            `xml:"Depth"`
	Azimuth           AzimuthType             `xml:"Azimuth"`
	Dip               DipType                 `xml:"Dip"`
	Type              []Type                  `xml:"Type"`
	SampleRate        SampleRateType          `xml:"SampleRate"`
	SampleRateRatio   SampleRateRatioType     `xml:"SampleRateRatio"`
	StorageFormat     string                  `xml:"StorageFormat"`
	ClockDrift        ClockDrift              `xml:"ClockDrift"`
	CalibrationUnits  UnitsType               `xml:"CalibrationUnits"`
	Sensor            EquipmentType           `xml:"Sensor"`
	PreAmplifier      EquipmentType           `xml:"PreAmplifier"`
	DataLogger        EquipmentType           `xml:"DataLogger"`
	Equipment         EquipmentType           `xml:"Equipment"`
	Response          ResponseType            `xml:"Response"`
}

type ClockDrift struct {
	Value      float64 `xml:",chardata"`
	Unit       string  `xml:"unit,attr"`
	PlusError  float64 `xml:"plusError,attr"`
	MinusError float64 `xml:"minusError,attr"`
}

type Coefficient struct {
	Value float64 `xml:",chardata"`
	FloatNoUnitType
	Number int `xml:"number,attr"`
}

// Response: coefficients for FIR filter. Laplace transforms or IIR
// filters can be expressed using type as well but the PolesAndZerosType should be used
// instead. Corresponds to SEED blockette 54.
type CoefficientsType struct {
	BaseFilterType
	CfTransferFunctionType CfTransferFunctionType `xml:"CfTransferFunctionType"`
	Numerator              []FloatType            `xml:"Numerator"`
	Denominator            []FloatType            `xml:"Denominator"`
}

type CommentType struct {
	Value              string       `xml:"Value"`
	BeginEffectiveTime xsdDateTime  `xml:"BeginEffectiveTime"`
	EndEffectiveTime   xsdDateTime  `xml:"EndEffectiveTime"`
	Author             []PersonType `xml:"Author"`
}

type DecimationType struct {
	InputSampleRate FrequencyType `xml:"InputSampleRate"`
	Factor          int           `xml:"Factor"`
	Offset          int           `xml:"Offset"`
	Delay           FloatType     `xml:"Delay"`
	Correction      FloatType     `xml:"Correction"`
}

// Instrument dip in degrees down from horizontal. Together azimuth and
// dip describe the direction of the sensitive axis of the instrument.
type DipType struct {
	Value      float64 `xml:",chardata"`
	Unit       string  `xml:"unit,attr"`
	PlusError  float64 `xml:"plusError,attr"`
	MinusError float64 `xml:"minusError,attr"`
}

// Extension of FloatType for distances, elevations, and
// depths.
type DistanceType struct {
	Value      float64 `xml:",chardata"`
	Unit       string  `xml:"unit,attr"`
	PlusError  float64 `xml:"plusError,attr"`
	MinusError float64 `xml:"minusError,attr"`
}

// Must match the pattern [\w\.\-_]+@[\w\.\-_]+
type EmailType string

type EquipmentType struct {
	ResourceId       string        `xml:"resourceId,attr"`
	Items            []string      `xml:",any"`
	Type             string        `xml:"Type"`
	Description      string        `xml:"Description"`
	Manufacturer     string        `xml:"Manufacturer"`
	Vendor           string        `xml:"Vendor"`
	Model            string        `xml:"Model"`
	SerialNumber     string        `xml:"SerialNumber"`
	InstallationDate xsdDateTime   `xml:"InstallationDate"`
	RemovalDate      xsdDateTime   `xml:"RemovalDate"`
	CalibrationDate  []xsdDateTime `xml:"CalibrationDate"`
}

type ExternalReferenceType struct {
	URI         string `xml:"URI"`
	Description string `xml:"Description"`
}

// Response: FIR filter. Corresponds to SEED blockette 61. FIR filters
// are also commonly documented using the CoefficientsType element.
type FIRType struct {
	BaseFilterType
	I                    int                    `xml:"i,attr"`
	Symmetry             Symmetry               `xml:"Symmetry"`
	NumeratorCoefficient []NumeratorCoefficient `xml:"NumeratorCoefficient"`
}

type FloatNoUnitType struct {
	Value      float64 `xml:",chardata"`
	PlusError  float64 `xml:"plusError,attr"`
	MinusError float64 `xml:"minusError,attr"`
}

// Representation of floating-point numbers used as
// measurements.
type FloatType struct {
	Value      float64 `xml:",chardata"`
	Unit       string  `xml:"unit,attr"`
	PlusError  float64 `xml:"plusError,attr"`
	MinusError float64 `xml:"minusError,attr"`
}

type FrequencyType struct {
	Value      float64 `xml:",chardata"`
	Unit       string  `xml:"unit,attr"`
	PlusError  float64 `xml:"plusError,attr"`
	MinusError float64 `xml:"minusError,attr"`
}

type GainType struct {
	Value     float64 `xml:"Value"`
	Frequency float64 `xml:"Frequency"`
}

// Base latitude type. Because of the limitations of schema, defining
// this type and then extending it to create the real latitude type is the only way to
// restrict values while adding datum as an attribute.
type LatitudeBaseType struct {
	Value      float64 `xml:",chardata"`
	Unit       string  `xml:"unit,attr"`
	PlusError  float64 `xml:"plusError,attr"`
	MinusError float64 `xml:"minusError,attr"`
}

// Type for latitude coordinate.
type LatitudeType struct {
	Value float64 `xml:",chardata"`
	LatitudeBaseType
	Datum string `xml:"datum,attr"`
}

type LogType struct {
	Entry []CommentType `xml:"Entry"`
}

type LongitudeBaseType struct {
	Value      float64 `xml:",chardata"`
	Unit       string  `xml:"unit,attr"`
	PlusError  float64 `xml:"plusError,attr"`
	MinusError float64 `xml:"minusError,attr"`
}

// Type for longitude coordinate.
type LongitudeType struct {
	Value float64 `xml:",chardata"`
	LongitudeBaseType
	Datum string `xml:"datum,attr"`
}

// This type represents the Network layer, all station metadata is
// contained within this element. The official name of the network or other descriptive
// information can be included in the Description element. The Network can contain 0 or
// more Stations.
type NetworkType struct {
	BaseNodeType
	TotalNumberStations    int           `xml:"TotalNumberStations"`
	SelectedNumberStations int           `xml:"SelectedNumberStations"`
	Station                []StationType `xml:"Station"`
}

// May be one of NOMINAL, CALCULATED
type NominalType string

type NumeratorCoefficient struct {
	Value float64 `xml:",chardata"`
	I     int     `xml:"i,attr"`
}

type Operator struct {
	Agency  []string     `xml:"Agency"`
	Contact []PersonType `xml:"Contact"`
	WebSite string       `xml:"WebSite"`
}

type PersonType struct {
	Name   []string          `xml:"Name"`
	Agency []string          `xml:"Agency"`
	Email  []EmailType       `xml:"Email"`
	Phone  []PhoneNumberType `xml:"Phone"`
}

// Must match the pattern [0-9]+-[0-9]+
type PhoneNumber string

type PhoneNumberType struct {
	Description string      `xml:"description,attr"`
	CountryCode int         `xml:"CountryCode"`
	AreaCode    int         `xml:"AreaCode"`
	PhoneNumber PhoneNumber `xml:"PhoneNumber"`
}

type PoleZeroType struct {
	Number    int             `xml:"number,attr"`
	Real      FloatNoUnitType `xml:"Real"`
	Imaginary FloatNoUnitType `xml:"Imaginary"`
}

// Response: complex poles and zeros. Corresponds to SEED blockette
// 53.
type PolesZerosType struct {
	BaseFilterType
	PzTransferFunctionType PzTransferFunctionType `xml:"PzTransferFunctionType"`
	NormalizationFactor    float64                `xml:"NormalizationFactor"`
	NormalizationFrequency FrequencyType          `xml:"NormalizationFrequency"`
	Zero                   []PoleZeroType         `xml:"Zero"`
	Pole                   []PoleZeroType         `xml:"Pole"`
}

// Response: expressed as a polynomial (allows non-linear sensors to be
// described). Corresponds to SEED blockette 62. Can be used to describe a stage of
// acquisition or a complete system.
type PolynomialType struct {
	BaseFilterType
	Number                  int               `xml:"number,attr"`
	ApproximationType       ApproximationType `xml:"ApproximationType"`
	FrequencyLowerBound     FrequencyType     `xml:"FrequencyLowerBound"`
	FrequencyUpperBound     FrequencyType     `xml:"FrequencyUpperBound"`
	ApproximationLowerBound float64           `xml:"ApproximationLowerBound"`
	ApproximationUpperBound float64           `xml:"ApproximationUpperBound"`
	MaximumError            float64           `xml:"MaximumError"`
	Coefficient             []Coefficient     `xml:"Coefficient"`
}

// May be one of LAPLACE (RADIANS/SECOND), LAPLACE (HERTZ), DIGITAL (Z-TRANSFORM)
type PzTransferFunctionType string

type ResponseListElementType struct {
	Frequency FrequencyType `xml:"Frequency"`
	Amplitude FloatType     `xml:"Amplitude"`
	Phase     AngleType     `xml:"Phase"`
}

// Response: list of frequency, amplitude and phase values. Corresponds
// to SEED blockette 55.
type ResponseListType struct {
	BaseFilterType
	ResponseListElement []ResponseListElementType `xml:"ResponseListElement"`
}

type ResponseStageType struct {
	Number       int              `xml:"number,attr"`
	ResourceId   string           `xml:"resourceId,attr"`
	Items        []string         `xml:",any"`
	PolesZeros   PolesZerosType   `xml:"PolesZeros"`
	Coefficients CoefficientsType `xml:"Coefficients"`
	ResponseList ResponseListType `xml:"ResponseList"`
	FIR          FIRType          `xml:"FIR"`
	Polynomial   PolynomialType   `xml:"Polynomial"`
	Decimation   DecimationType   `xml:"Decimation"`
	StageGain    GainType         `xml:"StageGain"`
}

type ResponseType struct {
	ResourceId            string              `xml:"resourceId,attr"`
	Items                 []string            `xml:",any"`
	InstrumentSensitivity SensitivityType     `xml:"InstrumentSensitivity"`
	InstrumentPolynomial  PolynomialType      `xml:"InstrumentPolynomial"`
	Stage                 []ResponseStageType `xml:"Stage"`
}

// May be one of open, closed, partial
type RestrictedStatusType string

type FDSNStationXML struct {
	XmlNs         string        `xml:"xmlns,attr" default:"http://www.fdsn.org/xml/station/1"`
	SchemaVersion float64       `xml:"schemaVersion,attr"`
	Items         []string      `xml:",any"`
	Source        string        `xml:"Source"`
	Sender        string        `xml:"Sender"`
	Module        string        `xml:"Module"`
	ModuleURI     string        `xml:"ModuleURI"`
	Created       xsdDateTime   `xml:"Created"`
	Network       []NetworkType `xml:"Network"`
}

type SampleRateRatioType struct {
	NumberSamples int `xml:"NumberSamples"`
	NumberSeconds int `xml:"NumberSeconds"`
}

// Sample rate in samples per second.
type SampleRateType struct {
	Value      float64 `xml:",chardata"`
	Unit       string  `xml:"unit,attr"`
	PlusError  float64 `xml:"plusError,attr"`
	MinusError float64 `xml:"minusError,attr"`
}

// A time value in seconds.
type SecondType struct {
	Value      float64 `xml:",chardata"`
	Unit       string  `xml:"unit,attr"`
	PlusError  float64 `xml:"plusError,attr"`
	MinusError float64 `xml:"minusError,attr"`
}

// Sensitivity and frequency ranges. The FrequencyRangeGroup is an
// optional construct that defines a pass band in Hertz (FrequencyStart and
// FrequencyEnd) in which the SensitivityValue is valid within the number of decibels
// specified in FrequencyDBVariation.
type SensitivityType struct {
	GainType
	InputUnits           UnitsType `xml:"InputUnits"`
	OutputUnits          UnitsType `xml:"OutputUnits"`
	FrequencyStart       float64   `xml:"FrequencyStart"`
	FrequencyEnd         float64   `xml:"FrequencyEnd"`
	FrequencyDBVariation float64   `xml:"FrequencyDBVariation"`
}

type SiteType struct {
	Items       []string `xml:",any"`
	Name        string   `xml:"Name"`
	Description string   `xml:"Description"`
	Town        string   `xml:"Town"`
	County      string   `xml:"County"`
	Region      string   `xml:"Region"`
	Country     string   `xml:"Country"`
}

// This type represents a Station epoch. It is common to only have a
// single station epoch with the station's creation and termination dates as the epoch
// start and end dates.
type StationType struct {
	BaseNodeType
	Latitude               LatitudeType            `xml:"Latitude"`
	Longitude              LongitudeType           `xml:"Longitude"`
	Elevation              DistanceType            `xml:"Elevation"`
	Site                   SiteType                `xml:"Site"`
	Vault                  string                  `xml:"Vault"`
	Geology                string                  `xml:"Geology"`
	Equipment              []EquipmentType         `xml:"Equipment"`
	Operator               []Operator              `xml:"Operator"`
	Agency                 []string                `xml:"Agency"`
	Contact                []PersonType            `xml:"Contact"`
	WebSite                string                  `xml:"WebSite"`
	CreationDate           xsdDateTime             `xml:"CreationDate"`
	TerminationDate        xsdDateTime             `xml:"TerminationDate"`
	TotalNumberChannels    int                     `xml:"TotalNumberChannels"`
	SelectedNumberChannels int                     `xml:"SelectedNumberChannels"`
	ExternalReference      []ExternalReferenceType `xml:"ExternalReference"`
	Channel                []ChannelType           `xml:"Channel"`
}

// May be one of NONE, EVEN, ODD
type Symmetry string

// May be one of TRIGGERED, CONTINUOUS, HEALTH, GEOPHYSICAL, WEATHER, FLAG, SYNTHESIZED, INPUT, EXPERIMENTAL, MAINTENANCE, BEAM
type Type string

type UnitsType struct {
	Name        string `xml:"Name"`
	Description string `xml:"Description"`
}

type VoltageType struct {
	Value      float64 `xml:",chardata"`
	Unit       string  `xml:"unit,attr"`
	PlusError  float64 `xml:"plusError,attr"`
	MinusError float64 `xml:"minusError,attr"`
}

type xsdDateTime time.Time

func (t *xsdDateTime) UnmarshalText(text []byte) error {
	return _unmarshalTime(text, (*time.Time)(t), "2006-01-02T15:04:05.999999999")
}
func (t *xsdDateTime) MarshalText() ([]byte, error) {
	return []byte((*time.Time)(t).Format("2006-01-02T15:04:05.999999999")), nil
}
func _unmarshalTime(text []byte, t *time.Time, format string) (err error) {
	s := string(bytes.TrimSpace(text))
	*t, err = time.Parse(format, s)
	if _, ok := err.(*time.ParseError); ok {
		*t, err = time.Parse(format+"Z07:00", s)
	}
	return err
}
