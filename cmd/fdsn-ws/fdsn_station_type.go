package main

import (
	"bytes"
	"time"

	"github.com/GeoNet/fdsn/internal/fdsn"
)

// May be one of MACLAURIN
type ApproximationType string

// CounterType represents integers greater than or equal to 0
type CounterType int

// DistanceType represents distance measurements in meters
type DistanceType struct {
	Value             float64  `xml:",chardata"`
	Unit              string   `xml:"unit,attr,omitempty"`
	PlusError         *float64 `xml:"plusError,attr,omitempty"`
	MinusError        *float64 `xml:"minusError,attr,omitempty"`
	MeasurementMethod string   `xml:"measurementMethod,attr,omitempty"`
}

// AzimuthType represents azimuth in degrees clockwise from north (0-360)
type AzimuthType struct {
	Value             float64  `xml:",chardata"`
	Unit              string   `xml:"unit,attr,omitempty"`
	PlusError         *float64 `xml:"plusError,attr,omitempty"`
	MinusError        *float64 `xml:"minusError,attr,omitempty"`
	MeasurementMethod string   `xml:"measurementMethod,attr,omitempty"`
}

// DipType represents dip in degrees, positive down from horizontal (-90 to +90)
type DipType struct {
	Value             float64  `xml:",chardata"`
	Unit              string   `xml:"unit,attr,omitempty"`
	PlusError         *float64 `xml:"plusError,attr,omitempty"`
	MinusError        *float64 `xml:"minusError,attr,omitempty"`
	MeasurementMethod string   `xml:"measurementMethod,attr,omitempty"`
}

// AngleType represents angle measurements in degrees (-360 to +360)
type AngleType struct {
	Value             float64  `xml:",chardata"`
	Unit              string   `xml:"unit,attr,omitempty"`
	PlusError         *float64 `xml:"plusError,attr,omitempty"`
	MinusError        *float64 `xml:"minusError,attr,omitempty"`
	MeasurementMethod string   `xml:"measurementMethod,attr,omitempty"`
}

// FrequencyType represents frequency measurements in Hertz
type FrequencyType struct {
	Value             float64  `xml:",chardata"`
	Unit              string   `xml:"unit,attr,omitempty"`
	PlusError         *float64 `xml:"plusError,attr,omitempty"`
	MinusError        *float64 `xml:"minusError,attr,omitempty"`
	MeasurementMethod string   `xml:"measurementMethod,attr,omitempty"`
}

// SampleRateType represents sample rate in samples per second
type SampleRateType struct {
	Value             float64  `xml:",chardata"`
	Unit              string   `xml:"unit,attr,omitempty"`
	PlusError         *float64 `xml:"plusError,attr,omitempty"`
	MinusError        *float64 `xml:"minusError,attr,omitempty"`
	MeasurementMethod string   `xml:"measurementMethod,attr,omitempty"`
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
	Code             string                `xml:"code,attr"`
	StartDate        xsdDateTime           `xml:"startDate,attr,omitempty"`
	EndDate          xsdDateTime           `xml:"endDate,attr,omitempty"`
	RestrictedStatus *RestrictedStatusType `xml:"restrictedStatus,attr,omitempty"`
	AlternateCode    string                `xml:"alternateCode,attr,omitempty"`
	HistoricalCode   string                `xml:"historicalCode,attr,omitempty"`
	SourceID         string                `xml:"sourceID,attr,omitempty"`
	Items            []string              `xml:",any,omitempty"`
	Description      string                `xml:"Description,omitempty"`
	Comment          []CommentType         `xml:"Comment,omitempty"`
	Identifier       []IdentifierType      `xml:"Identifier,omitempty"`
	DataAvailability *DataAvailabilityType `xml:"DataAvailability,omitempty"`
}

type IdentifierType struct {
	Type  string `xml:"type,attr,omitempty"`
	Value string `xml:",chardata"`
}

type DataAvailabilityType struct {
	Extent *DataAvailabilityExtentType `xml:"Extent,omitempty"`
	Span   []DataAvailabilitySpanType  `xml:"Span,omitempty"`
}

type DataAvailabilityExtentType struct {
	Start xsdDateTime `xml:"start,attr"`
	End   xsdDateTime `xml:"end,attr"`
}

type DataAvailabilitySpanType struct {
	Start           xsdDateTime `xml:"start,attr"`
	End             xsdDateTime `xml:"end,attr"`
	NumberSegments  int         `xml:"numberSegments,attr"`
	MaximumTimeTear float64     `xml:"maximumTimeTear,attr,omitempty"`
}

// May be one of ANALOG (RADIANS/SECOND), ANALOG (HERTZ), DIGITAL
type CfTransferFunctionType string

// Equivalent to SEED blockette 52 and parent element for the related the
// response blockettes.
type ChannelType struct {
	BaseNodeType
	LocationCode      string                  `xml:"locationCode,attr"`
	ExternalReference []ExternalReferenceType `xml:"ExternalReference,omitempty"`
	Latitude          LatitudeType            `xml:"Latitude"`
	Longitude         LongitudeType           `xml:"Longitude"`
	Elevation         DistanceType            `xml:"Elevation"`
	Depth             DistanceType            `xml:"Depth"`
	Azimuth           *AzimuthType            `xml:"Azimuth,omitempty"`
	Dip               *DipType                `xml:"Dip,omitempty"`
	WaterLevel        *FloatType              `xml:"WaterLevel,omitempty"`
	Type              []Type                  `xml:"Type,omitempty"`
	SampleRate        *SampleRateType         `xml:"SampleRate,omitempty"`
	SampleRateRatio   *SampleRateRatioType    `xml:"SampleRateRatio,omitempty"`
	ClockDrift        *FloatType              `xml:"ClockDrift,omitempty"`
	CalibrationUnits  *UnitsType              `xml:"CalibrationUnits,omitempty"`
	Sensor            *EquipmentType          `xml:"Sensor,omitempty"`
	PreAmplifier      *EquipmentType          `xml:"PreAmplifier,omitempty"`
	DataLogger        *EquipmentType          `xml:"DataLogger,omitempty"`
	Equipment         []EquipmentType         `xml:"Equipment,omitempty"`
	Response          *ResponseType           `xml:"Response,omitempty"`
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
	Id                 *int         `xml:"id,attr,omitempty"`
	Value              string       `xml:"Value,omitempty"`
	BeginEffectiveTime xsdDateTime  `xml:"BeginEffectiveTime,omitempty"`
	EndEffectiveTime   xsdDateTime  `xml:"EndEffectiveTime,omitempty"`
	Author             []PersonType `xml:"Author,omitempty"`
	Subject            string       `xml:"subject,attr,omitempty"`
}

type DecimationType struct {
	InputSampleRate FrequencyType `xml:"InputSampleRate"`
	Factor          int           `xml:"Factor"`
	Offset          int           `xml:"Offset"`
	Delay           FloatType     `xml:"Delay"`
	Correction      FloatType     `xml:"Correction"`
}

// EmailType must match pattern [\w\.\-_]+@[\w\.\-_]+
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
	SchemaVersion float64       `xml:"schemaVersion,attr"`
	Items         []string      `xml:",any,omitempty"`
	Source        string        `xml:"Source"`
	Sender        string        `xml:"Sender,omitempty"`
	Module        string        `xml:"Module,omitempty"`
	ModuleURI     string        `xml:"ModuleURI,omitempty"`
	Created       xsdDateTime   `xml:"Created"`
	Network       []NetworkType `xml:"Network"`
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
	Value             float64  `xml:",chardata"`
	PlusError         *float64 `xml:"plusError,attr,omitempty"`
	MinusError        *float64 `xml:"minusError,attr,omitempty"`
	MeasurementMethod string   `xml:"measurementMethod,attr,omitempty"`
}

// Representation of floating-point numbers used as
// measurements.
type FloatType struct {
	Value             float64  `xml:",chardata"`
	Unit              string   `xml:"unit,attr,omitempty"`
	PlusError         *float64 `xml:"plusError,attr,omitempty"`
	MinusError        *float64 `xml:"minusError,attr,omitempty"`
	MeasurementMethod string   `xml:"measurementMethod,attr,omitempty"`
}

type GainType struct {
	Value     float64  `xml:"Value,omitempty"`
	Frequency *float64 `xml:"Frequency,omitempty"`
}

// Base latitude type. Because of the limitations of schema, defining
// this type and then extending it to create the real latitude type is the only way to
// restrict values while adding datum as an attribute.

// Type for latitude coordinate.
type LatitudeType struct {
	Value float64 `xml:",chardata"`
	FloatType
	Datum string `xml:"datum,attr,omitempty"`
}

//nolint:deadcode,unused	// Struct based on FDSN spec so keep it
type LogType struct {
	Entry []CommentType `xml:"Entry,omitempty"`
}

// Type for longitude coordinate.
type LongitudeType struct {
	Value float64 `xml:",chardata"`
	FloatType
	Datum string `xml:"datum,attr,omitempty"`
}

// This type represents the Network layer, all station metadata is
// contained within this element. The official name of the network or other descriptive
// information can be included in the Description element. The Network can contain 0 or
// more Stations.
type NetworkType struct {
	BaseNodeType
	Operator               []Operator    `xml:"Operator,omitempty"`
	TotalNumberStations    CounterType   `xml:"TotalNumberStations,omitempty"`
	SelectedNumberStations CounterType   `xml:"SelectedNumberStations,omitempty"`
	Station                []StationType `xml:"Station,omitempty"`
}

// May be one of NOMINAL, CALCULATED
//
//nolint:deadcode,unused	// Struct based on FDSN spec so keep it
type NominalType string

type NumeratorCoefficient struct {
	Value float64 `xml:",chardata"`
	I     *int    `xml:"i,attr,omitempty"`
}

type Operator struct {
	Agency  string       `xml:"Agency"`
	Contact []PersonType `xml:"Contact,omitempty"`
	WebSite string       `xml:"WebSite,omitempty"`
}

type PersonType struct {
	Name   []string          `xml:"Name,omitempty"`
	Agency []string          `xml:"Agency,omitempty"`
	Email  []EmailType       `xml:"Email,omitempty"`
	Phone  []PhoneNumberType `xml:"Phone,omitempty"`
}

// PhoneNumber must match pattern [0-9]+-[0-9]+
type PhoneNumber string

type PhoneNumberType struct {
	Description string      `xml:"description,attr,omitempty"`
	CountryCode *int        `xml:"CountryCode,omitempty"`
	AreaCode    int         `xml:"AreaCode"`
	PhoneNumber PhoneNumber `xml:"PhoneNumber"`
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
	NormalizationFrequency *FloatType              `xml:"NormalizationFrequency,omitempty"`
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
	FrequencyLowerBound     *FloatType         `xml:"FrequencyLowerBound,omitempty"`
	FrequencyUpperBound     *FloatType         `xml:"FrequencyUpperBound,omitempty"`
	ApproximationLowerBound *float64           `xml:"ApproximationLowerBound,omitempty"`
	ApproximationUpperBound *float64           `xml:"ApproximationUpperBound,omitempty"`
	MaximumError            *float64           `xml:"MaximumError,omitempty"`
	Coefficient             []Coefficient      `xml:"Coefficient,omitempty"`
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

// RestrictedStatusType: open, closed, partial
type RestrictedStatusType string

const (
	RestrictedStatusOpen    RestrictedStatusType = "open"
	RestrictedStatusClosed  RestrictedStatusType = "closed"
	RestrictedStatusPartial RestrictedStatusType = "partial"
)

type SampleRateRatioType struct {
	NumberSamples *int `xml:"NumberSamples,omitempty"`
	NumberSeconds *int `xml:"NumberSeconds,omitempty"`
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
	Name        string   `xml:"Name"`
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
	Latitude               LatitudeType            `xml:"Latitude"`
	Longitude              LongitudeType           `xml:"Longitude"`
	Elevation              DistanceType            `xml:"Elevation"`
	Site                   SiteType                `xml:"Site"`
	WaterLevel             *FloatType              `xml:"WaterLevel,omitempty"`
	Vault                  string                  `xml:"Vault,omitempty"`
	Geology                string                  `xml:"Geology,omitempty"`
	Equipment              []EquipmentType         `xml:"Equipment,omitempty"`
	Operator               []Operator              `xml:"Operator,omitempty"`
	CreationDate           xsdDateTime             `xml:"CreationDate,omitempty"`
	TerminationDate        xsdDateTime             `xml:"TerminationDate,omitempty"`
	TotalNumberChannels    CounterType             `xml:"TotalNumberChannels,omitempty"`
	SelectedNumberChannels int                     `xml:"SelectedNumberChannels,omitempty"`
	ExternalReference      []ExternalReferenceType `xml:"ExternalReference,omitempty"`
	Channel                []ChannelType           `xml:"Channel,omitempty"`
}

// Symmetry: NONE, EVEN, ODD
type Symmetry string

const (
	SymmetryNone Symmetry = "NONE"
	SymmetryEven Symmetry = "EVEN"
	SymmetryOdd  Symmetry = "ODD"
)

// Channel types: TRIGGERED, CONTINUOUS, HEALTH, GEOPHYSICAL, WEATHER, FLAG, SYNTHESIZED, INPUT, EXPERIMENTAL, MAINTENANCE, BEAM
type Type string

const (
	TypeTriggered    Type = "TRIGGERED"
	TypeContinuous   Type = "CONTINUOUS"
	TypeHealth       Type = "HEALTH"
	TypeGeophysical  Type = "GEOPHYSICAL"
	TypeWeather      Type = "WEATHER"
	TypeFlag         Type = "FLAG"
	TypeSynthesized  Type = "SYNTHESIZED"
	TypeInput        Type = "INPUT"
	TypeExperimental Type = "EXPERIMENTAL"
	TypeMaintenance  Type = "MAINTENANCE"
	TypeBeam         Type = "BEAM"
)

type UnitsType struct {
	Name        string `xml:"Name,omitempty"`
	Description string `xml:"Description,omitempty"`
}

type xsdDateTime time.Time

func (t *xsdDateTime) UnmarshalText(text []byte) error {
	return _unmarshalTime(text, (*time.Time)(t))
}
func (t *xsdDateTime) MarshalText() ([]byte, error) {
	return []byte((*time.Time)(t).UTC().Format(time.RFC3339Nano)), nil
}

// Helper methods for backward compatibility
func (c CounterType) ToInt() int {
	return int(c)
}

func IntToCounterType(i int) *CounterType {
	c := CounterType(i)
	return &c
}
func _unmarshalTime(text []byte, t *time.Time) (err error) {
	s := string(bytes.TrimSpace(text))

	// Try RFC3339Nano first (new format)
	*t, err = time.Parse(time.RFC3339Nano, s)
	if err == nil {
		return nil
	}

	// Try legacy format for backward compatibility
	*t, err = time.Parse("2006-01-02T15:04:05.999999999", s)
	if err == nil {
		return nil
	}

	// Try legacy format with timezone
	*t, err = time.Parse("2006-01-02T15:04:05.999999999Z07:00", s)
	return err
}

// for decoder.Decode
func (t *Time) UnmarshalText(text []byte) (err error) {
	tm, err := fdsn.UnmarshalTime(text)
	if err != nil {
		return err
	}
	t.Time = tm
	return nil
}
