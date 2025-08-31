package ms

import (
	"encoding/binary"
)

const (
	BlocketteHeaderSize = 4
	Blockette1000Size   = 4
	Blockette1001Size   = 4
)

// BlocketteHeader stores the header of each miniseed blockette.
type BlocketteHeader struct {
	BlocketteType uint16
	NextBlockette uint16 // Byte of next blockette, 0 if last blockette
}

// DecodeBlocketteHeader returns a BlocketteHeader from a byte slice.
func DecodeBlocketteHeader(data []byte) BlocketteHeader {
	var h [BlocketteHeaderSize]byte

	copy(h[:], data)

	return BlocketteHeader{
		BlocketteType: binary.BigEndian.Uint16(h[0:2]),
		NextBlockette: binary.BigEndian.Uint16(h[2:4]),
	}
}

// EncodeBlocketteHeader converts a BlocketteHeader into a byte slice.
func EncodeBlocketteHeader(hdr BlocketteHeader) []byte {
	var b [BlocketteHeaderSize]byte

	binary.BigEndian.PutUint16(b[0:2], hdr.BlocketteType)
	binary.BigEndian.PutUint16(b[2:4], hdr.NextBlockette)

	h := make([]byte, BlocketteHeaderSize)
	copy(h[0:BlocketteHeaderSize], b[:])

	return h
}

// Unmarshal converts a byte slice into the BlocketteHeader
func (h *BlocketteHeader) Unmarshal(data []byte) error {
	*h = DecodeBlocketteHeader(data)
	return nil
}

// Marshal converts a BlocketteHeader into a byte slice.
func (h BlocketteHeader) Marshal() ([]byte, error) {
	return EncodeBlocketteHeader(h), nil
}

// Blockette1000 is a Data Only Seed Blockette (excluding header).
type Blockette1000 struct {
	Encoding     uint8
	WordOrder    uint8
	RecordLength uint8
	Reserved     uint8
}

// DecodeBlockette1000 returns a Blockette1000 from a byte slice.
func DecodeBlockette1000(data []byte) Blockette1000 {
	var b [Blockette1000Size]byte

	copy(b[:], data)

	return Blockette1000{
		Encoding:     b[0],
		WordOrder:    b[1],
		RecordLength: b[2],
		Reserved:     b[3],
	}
}

// EncodeBlockette1000 converts a Blockette1000 into a byte slice.
func EncodeBlockette1000(blk Blockette1000) []byte {
	var b [Blockette1000Size]byte

	b[0] = uint8(blk.Encoding)
	b[1] = uint8(blk.WordOrder)
	b[2] = uint8(blk.RecordLength)
	b[3] = uint8(blk.Reserved)

	d := make([]byte, Blockette1000Size)
	copy(d[0:Blockette1000Size], b[:])

	return d
}

// Unmarshal converts a byte slice into the Blockette1000
func (b *Blockette1000) Unmarshal(data []byte) error {
	*b = DecodeBlockette1000(data)
	return nil
}

// Marshal converts a Blockette1000 into a byte slice.
func (b Blockette1000) Marshal() ([]byte, error) {
	return EncodeBlockette1000(b), nil
}

// Blockette1001 is a "Data Extension Blockette" (excluding header).
type Blockette1001 struct {
	TimingQuality uint8
	MicroSec      int8 //Increased accuracy for starttime
	Reserved      uint8
	FrameCount    uint8
}

// DecodeBlockette1001 returns a Blockette1001 from a byte slice.
func DecodeBlockette1001(data []byte) Blockette1001 {
	var b [Blockette1001Size]byte

	copy(b[:], data)

	return Blockette1001{
		TimingQuality: b[0],
		MicroSec:      int8(b[1]),
		Reserved:      b[2],
		FrameCount:    b[3],
	}
}

func EncodeBlockette1001(blk Blockette1001) []byte {
	var b [Blockette1001Size]byte

	b[0] = blk.TimingQuality
	b[1] = uint8(blk.MicroSec) //nolint:gosec
	b[2] = blk.Reserved
	b[3] = blk.FrameCount

	d := make([]byte, Blockette1000Size)
	copy(d[0:Blockette1000Size], b[:])

	return d
}

// Unmarshal converts a byte slice into the Blockette1001
func (b *Blockette1001) Unmarshal(data []byte) error {
	*b = DecodeBlockette1001(data)
	return nil
}

// Marshal converts a Blockette1001 into a byte slice.
func (b Blockette1001) Marshal() ([]byte, error) {
	return EncodeBlockette1001(b), nil
}
