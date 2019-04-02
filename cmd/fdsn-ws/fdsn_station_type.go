package main

import (
	"bytes"
	"time"
)

type AngleType struct {
	Value      float64  `xml:",chardata"`
	Unit       string   `xml:"unit,attr,omitempty"`
	PlusError  *float64 `xml:"plusError,attr,omitempty"`
	MinusError *float64 `xml:"minusError,attr,omitempty"`
}

// May be one of MACLAURIN
type ApproximationType string

// Instrument azimuth, degrees clockwise from North.
type AzimuthType struct {
	Value      float64  `xml:",chardata"`
	Unit       string   `xml:"unit,attr,omitempty"`
	PlusError  *float64 `xml:"plusError,attr,omitempty"`
	MinusError *float64 `xml:"minusError,attr,omitempty"`
}

type BaseFilterType struct {
	ResourceId  string     `xml:"resourceId,attr,omitempty"`
	Name        string     `xml:"name,attr,omitempty"`
	Items       []string   `xml:",any,omitempty"`
	Description string     `xml:"Description,omitempty"`
	InputUnits  *UnitsType `xml:"InputUnits,omitempty"`
	OutputUnits *UnitsType `xml:"OutputUnits,omitempty"`
}

type BaseNodeType struct {
	Code             string                `xml:"code,attr,omitempty"`
	StartDate        xsdDateTime           `xml:"startDate,attr,omitempty"`
	EndDate          xsdDateTime           `xml:"endDate,attr,omitempty"`
	RestrictedStatus *RestrictedStatusType `xml:"restrictedStatus,attr,omitempty"`
	AlternateCode    string                `xml:"alternateCode,attr,omitempty"`
	HistoricalCode   string                `xml:"historicalCode,attr,omitempty"`
	Items            []string              `xml:",any,omitempty"`
	Description      string                `xml:"Description,omitempty"`
	Comment          []CommentType         `xml:"Comment,omitempty"`
}

// May be one of ANALOG (RADIANS/SECOND), ANALOG (HERTZ), DIGITAL
type CfTransferFunctionType string

// Equivalent to SEED blockette 52 and parent element for the related the
// response blockettes.
type ChannelType struct {
	BaseNodeType
	Unit              string                  `xml:"unit,attr,omitempty"`
	LocationCode      string                  `xml:"locationCode,attr,omitempty"`
	ExternalReference []ExternalReferenceType `xml:"ExternalReference,omitempty"`
	Latitude          *LatitudeType           `xml:"Latitude,omitempty"`
	Longitude         *LongitudeType          `xml:"Longitude,omitempty"`
	Elevation         *DistanceType           `xml:"Elevation,omitempty"`
	Depth             *DistanceType           `xml:"Depth,omitempty"`
	Azimuth           *AzimuthType            `xml:"Azimuth,omitempty"`
	Dip               *DipType                `xml:"Dip,omitempty"`
	Type              []Type                  `xml:"Type,omitempty"`
	SampleRate        *SampleRateType         `xml:"SampleRate,omitempty"`
	SampleRateRatio   *SampleRateRatioType    `xml:"SampleRateRatio,omitempty"`
	StorageFormat     string                  `xml:"StorageFormat,omitempty"`
	ClockDrift        *ClockDrift             `xml:"ClockDrift,omitempty"`
	CalibrationUnits  *UnitsType              `xml:"CalibrationUnits,omitempty"`
	Sensor            *EquipmentType          `xml:"Sensor,omitempty"`
	PreAmplifier      *EquipmentType          `xml:"PreAmplifier,omitempty"`
	DataLogger        *EquipmentType          `xml:"DataLogger,omitempty"`
	Equipment         *EquipmentType          `xml:"Equipment,omitempty"`
	Response          *ResponseType           `xml:"Response,omitempty"`
}

type ClockDrift struct {
	Value      float64  `xml:",chardata"`
	Unit       string   `xml:"unit,attr,omitempty"`
	PlusError  *float64 `xml:"plusError,attr,omitempty"`
	MinusError *float64 `xml:"minusError,attr,omitempty"`
}

type Coefficient struct {
	Value float64 `xml:",chardata"`
	FloatNoUnitType
	Number *int `xml:"number,attr,omitempty"`
}

// Response: coefficients for FIR filter. Laplace transforms or IIR
// filters can be expressed using type as well but the PolesAndZerosType should be used
// instead. Corresponds to SEED blockette 54.
type CoefficientsType struct {
	BaseFilterType
	CfTransferFunctionType *CfTransferFunctionType `xml:"CfTransferFunctionType,omitempty"`
	Numerator              []FloatType             `xml:"Numerator,omitempty"`
	Denominator            []FloatType             `xml:"Denominator,omitempty"`
}

type CommentType struct {
	Value              string       `xml:"Value,omitempty"`
	BeginEffectiveTime xsdDateTime  `xml:"BeginEffectiveTime,omitempty"`
	EndEffectiveTime   xsdDateTime  `xml:"EndEffectiveTime,omitempty"`
	Author             []PersonType `xml:"Author,omitempty"`
}

type DecimationType struct {
	InputSampleRate *FrequencyType `xml:"InputSampleRate,omitempty"`
	Factor          *int           `xml:"Factor,omitempty"`
	Offset          *int           `xml:"Offset,omitempty"`
	Delay           *FloatType     `xml:"Delay,omitempty"`
	Correction      *FloatType     `xml:"Correction,omitempty"`
}

// Instrument dip in degrees down from horizontal. Together azimuth and
// dip describe the direction of the sensitive axis of the instrument.
type DipType struct {
	Value      float64  `xml:",chardata"`
	Unit       string   `xml:"unit,attr,omitempty"`
	PlusError  *float64 `xml:"plusError,attr,omitempty"`
	MinusError *float64 `xml:"minusError,attr,omitempty"`
}

// Extension of FloatType for distances, elevations, and
// depths.
type DistanceType struct {
	Value      float64  `xml:",chardata"`
	Unit       string   `xml:"unit,attr,omitempty"`
	PlusError  *float64 `xml:"plusError,attr,omitempty"`
	MinusError *float64 `xml:"minusError,attr,omitempty"`
}

// Must match the pattern [\w\.\-_]+@[\w\.\-_]+
type EmailType string

type EquipmentType struct {
	ResourceId       string        `xml:"resourceId,attr,omitempty"`
	Items            []string      `xml:",any,omitempty"`
	Type             string        `xml:"Type,omitempty"`
	Description      string        `xml:"Description,omitempty"`
	Manufacturer     string        `xml:"Manufacturer,omitempty"`
	Vendor           string        `xml:"Vendor,omitempty"`
	Model            string        `xml:"Model,omitempty"`
	SerialNumber     string        `xml:"SerialNumber,omitempty"`
	InstallationDate xsdDateTime   `xml:"InstallationDate,omitempty"`
	RemovalDate      xsdDateTime   `xml:"RemovalDate,omitempty"`
	CalibrationDate  []xsdDateTime `xml:"CalibrationDate,omitempty"`
}

type ExternalReferenceType struct {
	URI         string `xml:"URI,omitempty"`
	Description string `xml:"Description,omitempty"`
}

type FDSNStationXML struct {
	SchemaVersion *float64      `xml:"schemaVersion,attr,omitempty"`
	Items         []string      `xml:",any,omitempty"`
	Source        string        `xml:"Source,omitempty"`
	Sender        string        `xml:"Sender,omitempty"`
	Module        string        `xml:"Module,omitempty"`
	ModuleURI     string        `xml:"ModuleURI,omitempty"`
	Created       xsdDateTime   `xml:"Created,omitempty"`
	Network       []NetworkType `xml:"Network,omitempty"`
	Xmlns         string        `xml:"xmlns,attr,omitempty"`
}

// Response: FIR filter. Corresponds to SEED blockette 61. FIR filters
// are also commonly documented using the CoefficientsType element.
type FIRType struct {
	BaseFilterType
	I                    *int                   `xml:"i,attr,omitempty"`
	Symmetry             *Symmetry              `xml:"Symmetry,omitempty"`
	NumeratorCoefficient []NumeratorCoefficient `xml:"NumeratorCoefficient,omitempty"`
}

type FloatNoUnitType struct {
	Value      float64  `xml:",chardata"`
	PlusError  *float64 `xml:"plusError,attr,omitempty"`
	MinusError *float64 `xml:"minusError,attr,omitempty"`
}

// Representation of floating-point numbers used as
// measurements.
type FloatType struct {
	Value      float64  `xml:",chardata"`
	Unit       string   `xml:"unit,attr,omitempty"`
	PlusError  *float64 `xml:"plusError,attr,omitempty"`
	MinusError *float64 `xml:"minusError,attr,omitempty"`
}

type FrequencyType struct {
	Value      float64  `xml:",chardata"`
	Unit       string   `xml:"unit,attr,omitempty"`
	PlusError  *float64 `xml:"plusError,attr,omitempty"`
	MinusError *float64 `xml:"minusError,attr,omitempty"`
}

type GainType struct {
	Value     float64  `xml:"Value,omitempty"`
	Frequency *float64 `xml:"Frequency,omitempty"`
}

// Base latitude type. Because of the limitations of schema, defining
// this type and then extending it to create the real latitude type is the only way to
// restrict values while adding datum as an attribute.
type LatitudeBaseType struct {
	Value      float64  `xml:",chardata"`
	Unit       string   `xml:"unit,attr,omitempty"`
	PlusError  *float64 `xml:"plusError,attr,omitempty"`
	MinusError *float64 `xml:"minusError,attr,omitempty"`
}

// Type for latitude coordinate.
type LatitudeType struct {
	Value float64 `xml:",chardata"`
	LatitudeBaseType
	Datum string `xml:"datum,attr,omitempty"`
}

//nolint:deadcode,unused	// Struct based on FDSN spec so keep it
type LogType struct {
	Entry []CommentType `xml:"Entry,omitempty"`
}

type LongitudeBaseType struct {
	Value      float64  `xml:",chardata"`
	Unit       string   `xml:"unit,attr,omitempty"`
	PlusError  *float64 `xml:"plusError,attr,omitempty"`
	MinusError *float64 `xml:"minusError,attr,omitempty"`
}

// Type for longitude coordinate.
type LongitudeType struct {
	Value float64 `xml:",chardata"`
	LongitudeBaseType
	Datum string `xml:"datum,attr,omitempty"`
}

// This type represents the Network layer, all station metadata is
// contained within this element. The official name of the network or other descriptive
// information can be included in the Description element. The Network can contain 0 or
// more Stations.
type NetworkType struct {
	BaseNodeType
	TotalNumberStations    int           `xml:"TotalNumberStations"`
	SelectedNumberStations int           `xml:"SelectedNumberStations"`
	Station                []StationType `xml:"Station,omitempty"`
}

// May be one of NOMINAL, CALCULATED
//nolint:deadcode,unused	// Struct based on FDSN spec so keep it
type NominalType string

type NumeratorCoefficient struct {
	Value float64 `xml:",chardata"`
	I     *int    `xml:"i,attr,omitempty"`
}

type Operator struct {
	Agency  []string     `xml:"Agency,omitempty"`
	Contact []PersonType `xml:"Contact,omitempty"`
	WebSite string       `xml:"WebSite,omitempty"`
}

type PersonType struct {
	Name   []string          `xml:"Name,omitempty"`
	Agency []string          `xml:"Agency,omitempty"`
	Email  []EmailType       `xml:"Email,omitempty"`
	Phone  []PhoneNumberType `xml:"Phone,omitempty"`
}

// Must match the pattern [0-9]+-[0-9]+
type PhoneNumber string

type PhoneNumberType struct {
	Description string       `xml:"description,attr,omitempty"`
	CountryCode *int         `xml:"CountryCode,omitempty"`
	AreaCode    *int         `xml:"AreaCode,omitempty"`
	PhoneNumber *PhoneNumber `xml:"PhoneNumber,omitempty"`
}

type PoleZeroType struct {
	Number    *int             `xml:"number,attr,omitempty"`
	Real      *FloatNoUnitType `xml:"Real,omitempty"`
	Imaginary *FloatNoUnitType `xml:"Imaginary,omitempty"`
}

// Response: complex poles and zeros. Corresponds to SEED blockette
// 53.
type PolesZerosType struct {
	BaseFilterType
	PzTransferFunctionType *PzTransferFunctionType `xml:"PzTransferFunctionType,omitempty"`
	NormalizationFactor    *float64                `xml:"NormalizationFactor,omitempty"`
	NormalizationFrequency *FrequencyType          `xml:"NormalizationFrequency,omitempty"`
	Zero                   []PoleZeroType          `xml:"Zero,omitempty"`
	Pole                   []PoleZeroType          `xml:"Pole,omitempty"`
}

// Response: expressed as a polynomial (allows non-linear sensors to be
// described). Corresponds to SEED blockette 62. Can be used to describe a stage of
// acquisition or a complete system.
type PolynomialType struct {
	BaseFilterType
	Number                  *int               `xml:"number,attr,omitempty"`
	ApproximationType       *ApproximationType `xml:"ApproximationType,omitempty"`
	FrequencyLowerBound     *FrequencyType     `xml:"FrequencyLowerBound,omitempty"`
	FrequencyUpperBound     *FrequencyType     `xml:"FrequencyUpperBound,omitempty"`
	ApproximationLowerBound *float64           `xml:"ApproximationLowerBound,omitempty"`
	ApproximationUpperBound *float64           `xml:"ApproximationUpperBound,omitempty"`
	MaximumError            *float64           `xml:"MaximumError,omitempty"`
	Coefficient             []Coefficient      `xml:"Coefficient,omitempty"`
}

// May be one of LAPLACE (RADIANS/SECOND), LAPLACE (HERTZ), DIGITAL (Z-TRANSFORM)
type PzTransferFunctionType string

type ResponseListElementType struct {
	Frequency *FrequencyType `xml:"Frequency,omitempty"`
	Amplitude *FloatType     `xml:"Amplitude,omitempty"`
	Phase     *AngleType     `xml:"Phase,omitempty"`
}

// Response: list of frequency, amplitude and phase values. Corresponds
// to SEED blockette 55.
type ResponseListType struct {
	BaseFilterType
	ResponseListElement []ResponseListElementType `xml:"ResponseListElement,omitempty"`
}

type ResponseStageType struct {
	Number       *int              `xml:"number,attr,omitempty"`
	ResourceId   string            `xml:"resourceId,attr,omitempty"`
	Items        []string          `xml:",any,omitempty"`
	PolesZeros   *PolesZerosType   `xml:"PolesZeros,omitempty"`
	Coefficients *CoefficientsType `xml:"Coefficients,omitempty"`
	ResponseList *ResponseListType `xml:"ResponseList,omitempty"`
	FIR          *FIRType          `xml:"FIR,omitempty"`
	Polynomial   *PolynomialType   `xml:"Polynomial,omitempty"`
	Decimation   *DecimationType   `xml:"Decimation,omitempty"`
	StageGain    *GainType         `xml:"StageGain,omitempty"`
}

type ResponseType struct {
	ResourceId            string              `xml:"resourceId,attr,omitempty"`
	Items                 []string            `xml:",any,omitempty"`
	InstrumentSensitivity *SensitivityType    `xml:"InstrumentSensitivity,omitempty"`
	InstrumentPolynomial  *PolynomialType     `xml:"InstrumentPolynomial,omitempty"`
	Stage                 []ResponseStageType `xml:"Stage,omitempty"`
}

// May be one of open, closed, partial
type RestrictedStatusType string

type SampleRateRatioType struct {
	NumberSamples *int `xml:"NumberSamples,omitempty"`
	NumberSeconds *int `xml:"NumberSeconds,omitempty"`
}

// Sample rate in samples per second.
type SampleRateType struct {
	Value      float64  `xml:",chardata"`
	Unit       string   `xml:"unit,attr,omitempty"`
	PlusError  *float64 `xml:"plusError,attr,omitempty"`
	MinusError *float64 `xml:"minusError,attr,omitempty"`
}

// A time value in seconds.
//nolint:deadcode,unused	// Struct based on FDSN spec so keep it
type SecondType struct {
	Value      float64  `xml:",chardata"`
	Unit       string   `xml:"unit,attr,omitempty"`
	PlusError  *float64 `xml:"plusError,attr,omitempty"`
	MinusError *float64 `xml:"minusError,attr,omitempty"`
}

// Sensitivity and frequency ranges. The FrequencyRangeGroup is an
// optional construct that defines a pass band in Hertz (FrequencyStart and
// FrequencyEnd) in which the SensitivityValue is valid within the number of decibels
// specified in FrequencyDBVariation.
type SensitivityType struct {
	GainType
	InputUnits           *UnitsType `xml:"InputUnits,omitempty"`
	OutputUnits          *UnitsType `xml:"OutputUnits,omitempty"`
	FrequencyStart       *float64   `xml:"FrequencyStart,omitempty"`
	FrequencyEnd         *float64   `xml:"FrequencyEnd,omitempty"`
	FrequencyDBVariation *float64   `xml:"FrequencyDBVariation,omitempty"`
}

type SiteType struct {
	Items       []string `xml:",any,omitempty"`
	Name        string   `xml:"Name,omitempty"`
	Description string   `xml:"Description,omitempty"`
	Town        string   `xml:"Town,omitempty"`
	County      string   `xml:"County,omitempty"`
	Region      string   `xml:"Region,omitempty"`
	Country     string   `xml:"Country,omitempty"`
}

// This type represents a Station epoch. It is common to only have a
// single station epoch with the station's creation and termination dates as the epoch
// start and end dates.
type StationType struct {
	BaseNodeType
	Latitude               *LatitudeType           `xml:"Latitude,omitempty"`
	Longitude              *LongitudeType          `xml:"Longitude,omitempty"`
	Elevation              *DistanceType           `xml:"Elevation,omitempty"`
	Site                   *SiteType               `xml:"Site,omitempty"`
	Vault                  string                  `xml:"Vault,omitempty"`
	Geology                string                  `xml:"Geology,omitempty"`
	Equipment              []EquipmentType         `xml:"Equipment,omitempty"`
	Operator               []Operator              `xml:"Operator,omitempty"`
	Agency                 []string                `xml:"Agency,omitempty"`
	Contact                []PersonType            `xml:"Contact,omitempty"`
	WebSite                string                  `xml:"WebSite,omitempty"`
	CreationDate           xsdDateTime             `xml:"CreationDate,omitempty"`
	TerminationDate        xsdDateTime             `xml:"TerminationDate,omitempty"`
	TotalNumberChannels    int                     `xml:"TotalNumberChannels"`
	SelectedNumberChannels int                     `xml:"SelectedNumberChannels"`
	ExternalReference      []ExternalReferenceType `xml:"ExternalReference,omitempty"`
	Channel                []ChannelType           `xml:"Channel,omitempty"`
}

// May be one of NONE, EVEN, ODD
type Symmetry string

// May be one of TRIGGERED, CONTINUOUS, HEALTH, GEOPHYSICAL, WEATHER, FLAG, SYNTHESIZED, INPUT, EXPERIMENTAL, MAINTENANCE, BEAM
type Type string

type UnitsType struct {
	Name        string `xml:"Name,omitempty"`
	Description string `xml:"Description,omitempty"`
}

//nolint:deadcode,unused	// Struct based on FDSN spec so keep it
type VoltageType struct {
	Value      float64  `xml:",chardata"`
	Unit       string   `xml:"unit,attr,omitempty"`
	PlusError  *float64 `xml:"plusError,attr,omitempty"`
	MinusError *float64 `xml:"minusError,attr,omitempty"`
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
