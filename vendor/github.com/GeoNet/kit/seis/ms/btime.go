package ms

import (
	"encoding/binary"
	"time"
)

// BTimeSize is the fixed size of an encoded BTime.
const BTimeSize = 10

// BTime is the SEED Representation of Time.
type BTime struct {
	Year   uint16
	Doy    uint16
	Hour   uint8
	Minute uint8
	Second uint8
	Unused byte
	S0001  uint16
}

// Time converts a BTime into a time.Time.
func (b BTime) Time() time.Time {
	return time.Date(
		int(b.Year),
		1,
		1,
		int(b.Hour),
		int(b.Minute),
		int(b.Second),
		int(b.S0001)*100000,
		time.UTC,
	).AddDate(0, 0, int(b.Doy-1))
}

// NewBTime builds a BTime from a time.Time.
func NewBTime(t time.Time) BTime {
	return BTime{
		Year:   uint16(t.Year()),                //nolint:gosec
		Doy:    uint16(t.YearDay()),             //nolint:gosec
		Hour:   uint8(t.Hour()),                 //nolint:gosec
		Minute: uint8(t.Minute()),               //nolint:gosec
		Second: uint8(t.Second()),               //nolint:gosec
		S0001:  uint16(t.Nanosecond() / 100000), //nolint:gosec
	}
}

// DecodeBTime returns a BTime from a byte slice.
func DecodeBTime(data []byte) BTime {
	var b [BTimeSize]byte

	copy(b[:], data)

	return BTime{
		Year:   binary.BigEndian.Uint16(b[0:2]),
		Doy:    binary.BigEndian.Uint16(b[2:4]),
		Hour:   b[4],
		Minute: b[5],
		Second: b[6],
		Unused: b[7],
		S0001:  binary.BigEndian.Uint16(b[8:10]),
	}
}

// EncodeBTime converts a BTime into a byte slice.
func EncodeBTime(at BTime) []byte {
	var b [BTimeSize]byte

	binary.BigEndian.PutUint16(b[0:2], at.Year)
	binary.BigEndian.PutUint16(b[2:4], at.Doy)
	b[4] = at.Hour
	b[5] = at.Minute
	b[6] = at.Second
	b[7] = at.Unused
	binary.BigEndian.PutUint16(b[8:10], at.S0001)

	t := make([]byte, BTimeSize)
	copy(t[0:BTimeSize], b[:])

	return t
}

func (b *BTime) Unmarshal(data []byte) error {
	*b = DecodeBTime(data)
	return nil
}

func (b BTime) Marshal() ([]byte, error) {
	return EncodeBTime(b), nil
}
