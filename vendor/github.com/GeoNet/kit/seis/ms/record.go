package ms

import (
	"fmt"
	"strings"
	"time"
)

type Encoding uint8

const (
	EncodingASCII      Encoding = 0
	EncodingInt32      Encoding = 3
	EncodingIEEEFloat  Encoding = 4
	EncodingIEEEDouble Encoding = 5
	EncodingSTEIM1     Encoding = 10
	EncodingSTEIM2     Encoding = 11
)

type WordOrder uint8

const (
	LittleEndian WordOrder = 0
	BigEndian    WordOrder = 1
)

type SampleType byte

const (
	UnknownType SampleType = 0
	ByteType    SampleType = 'a'
	IntegerType SampleType = 'i'
	FloatType   SampleType = 'f'
	DoubleType  SampleType = 'd'
)

type Record struct {
	RecordHeader

	B1000 Blockette1000 //If Present
	B1001 Blockette1001 //If Present

	Data []byte
}

// NewMSRecord decodes and unpacks the record samples from a byte slice and returns a Record pointer,
// or an empty pointer and an error if it could not be decoded.
func NewRecord(buf []byte) (*Record, error) {

	var r Record
	if err := r.Unpack(buf); err != nil {
		return nil, err
	}

	return &r, nil
}

// String implements the Stringer interface and provides a short summary of the miniseed record header.
func (m Record) String() string {
	var parts []string

	parts = append(parts, m.SrcName(false))
	parts = append(parts, fmt.Sprintf("%06d", m.SeqNumber()))
	parts = append(parts, string(m.DataQualityIndicator))
	parts = append(parts, fmt.Sprintf("%d", m.BlockSize()))
	parts = append(parts, fmt.Sprintf("%d samples", m.NumberOfSamples))
	parts = append(parts, fmt.Sprintf("%g Hz", m.SampleRate()))
	parts = append(parts, m.StartTime().Format("2006,002,15:04:05.000000"))

	return strings.Join(parts, ", ")
}

// StartTime returns the calculated time of the first sample.
func (m Record) StartTime() time.Time {
	t := m.RecordHeader.StartTime()

	if m.B1001.MicroSec != 0 {
		t = t.Add(time.Microsecond * time.Duration(m.B1001.MicroSec))
	}

	return t
}

// EndTime returns the calculated time of the last sample.
func (m Record) EndTime() time.Time {
	var d time.Duration

	if sc, sr := m.SampleCount(), m.SampleRate(); sc > 0 && sr > 0 {
		d = time.Duration(m.SampleCount()-1) * time.Duration(float64(time.Second)/m.SampleRate()+0.5)
	}

	return m.StartTime().Add(d)
}

// PacketSize returns the length of the packet
func (m Record) BlockSize() int {
	if n := int(m.B1000.RecordLength); n > 0 {
		return 1 << n
	}
	return 0
}

// Encoding returns the miniseed data format encoding.
func (m Record) Encoding() Encoding {
	return Encoding(m.B1000.Encoding)
}

// ByteOrder returns the miniseed data byte order.
func (m Record) ByteOrder() WordOrder {
	switch m.B1000.WordOrder {
	case 0:
		return LittleEndian
	default:
		return BigEndian
	}
}

// SampleType returns the type of samples decoded, or UnknownType if no data has been decoded.
func (m Record) SampleType() SampleType {
	switch Encoding(m.B1000.Encoding) {
	case EncodingASCII:
		return ByteType
	case EncodingInt32:
		return IntegerType
	case EncodingIEEEFloat:
		return FloatType
	case EncodingIEEEDouble:
		return DoubleType
	case EncodingSTEIM1:
		return IntegerType
	case EncodingSTEIM2:
		return IntegerType
	default:
		return UnknownType
	}
}
