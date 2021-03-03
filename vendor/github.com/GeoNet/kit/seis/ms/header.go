package ms

import (
	"encoding/binary"
	"fmt"
	//TODO: needs v 1.15 "hash/maphash"
	"strconv"
	"strings"
	"time"
)

// RecordHeaderSize is the miniseed block fixed header length.
const RecordHeaderSize = 48

type RecordHeader struct {
	SequenceNumber [6]byte // ASCII String representing a 6 digit number

	DataQualityIndicator byte // ASCII: D, R, Q or M
	ReservedByte         byte // ASCII: Space

	// These are ascii strings
	StationIdentifier  [5]byte // ASCII: Left justify and pad with spaces
	LocationIdentifier [2]byte // ASCII: Left justify and pad with spaces
	ChannelIdentifier  [3]byte // ASCII: Left justify and pad with spaces
	NetworkIdentifier  [2]byte // ASCII: Left justify and pad with spaces

	RecordStartTime      BTime  // Start time of record
	NumberOfSamples      uint16 // Number of Samples in the data block which may or may not be unpacked.
	SampleRateFactor     int16  // >0: Samples/Second <0: Second/Samples 0: Seconds/Sample, ASCII/OPAQUE DATA records
	SampleRateMultiplier int16  // >0: Multiplication Factor <0: Division Factor

	// Flags are bit masks
	ActivityFlags    byte
	IOAndClockFlags  byte
	DataQualityFlags byte

	NumberOfBlockettesThatFollow uint8 // Total number of blockettes that follow

	TimeCorrection  int32  // 0.0001 second units
	BeginningOfData uint16 // Offset in bytes to the beginning of data.
	FirstBlockette  uint16 // Offset in bytes to the first data blockette in the data record.
}

func (h RecordHeader) SeqNumber() int {
	s := string(h.SequenceNumber[:])
	if no, err := strconv.ParseInt(s, 10, 32); err == nil {
		return int(no)
	}
	return 0
}
func (h *RecordHeader) SetSeqNumber(no int) {
	copy(h.SequenceNumber[:], []byte(fmt.Sprintf("%06d", no)))
}

func (h RecordHeader) Station() string {
	return strings.TrimSpace(string(h.StationIdentifier[:]))
}
func (h *RecordHeader) SetStation(s string) {
	for len(s) < 5 {
		s += " "
	}
	copy(h.StationIdentifier[:], s)
}
func (h RecordHeader) Location() string {
	return strings.TrimSpace(string(h.LocationIdentifier[:]))
}
func (h *RecordHeader) SetLocation(s string) {
	for len(s) < 2 {
		s += " "
	}
	copy(h.LocationIdentifier[:], s)
}
func (h RecordHeader) Channel() string {
	return strings.TrimSpace(string(h.ChannelIdentifier[:]))
}
func (h *RecordHeader) SetChannel(s string) {
	for len(s) < 3 {
		s += " "
	}
	copy(h.ChannelIdentifier[:], s)
}
func (h RecordHeader) Network() string {
	return strings.TrimSpace(string(h.NetworkIdentifier[:]))
}
func (h *RecordHeader) SetNetwork(s string) {
	for len(s) < 2 {
		s += " "
	}
	copy(h.NetworkIdentifier[:], s)
}
func (h RecordHeader) SrcName(quality bool) string {
	if quality {
		return strings.Join([]string{h.Network(), h.Station(), h.Location(), h.Channel(), string(h.DataQualityIndicator)}, "_")
	}
	return strings.Join([]string{h.Network(), h.Station(), h.Location(), h.Channel()}, "_")
}

func (h *RecordHeader) SetStartTime(t time.Time) {
	h.RecordStartTime = NewBTime(t)
}

func (h *RecordHeader) SetCorrection(correction time.Duration, applied bool) {
	switch applied {
	case true:
		h.ActivityFlags = setBit(h.ActivityFlags, 1)
	default:
		h.ActivityFlags = clearBit(h.ActivityFlags, 1)
	}
	h.TimeCorrection = int32(correction / (100 * time.Microsecond))
}

func (h RecordHeader) Correction() time.Duration {
	return 100 * time.Microsecond * time.Duration(h.TimeCorrection)
}

func (h RecordHeader) StartTime() time.Time {

	start := h.RecordStartTime.Time()

	// Check for a header time correction
	if !isBitSet(h.ActivityFlags, 1) { // bit 1 indicates whether time correction has been applied
		start = start.Add(100 * time.Microsecond * time.Duration(h.TimeCorrection))
	}

	return start
}

// SampleRate returns the decoded header sampling rate in samples per second.
func (h RecordHeader) SampleRate() float64 {
	return sampleRate(int(h.SampleRateFactor), int(h.SampleRateMultiplier))
}

// SampleCount returns the number of samples in the record, independent of whether they are decoded or not.
func (h RecordHeader) SampleCount() int {
	return int(h.NumberOfSamples)
}

// SamplePeriod converts the sample rate into a time interval, or zero.
func (h RecordHeader) SamplePeriod() time.Duration {
	return samplePeriod(int(h.SampleRateFactor), int(h.SampleRateMultiplier))
}

// IsValid performs a simple consistency check of the RecordHeader contents.
func (h RecordHeader) IsValid() bool {
	for _, b := range h.SequenceNumber {
		if !((b >= '0' && b <= '9') || (b == ' ') || (b == 0)) {
			return false
		}
	}

	if q := h.DataQualityIndicator; !(q == 'D' || q == 'R' || q == 'M' || q == 'Q') {
		return false
	}

	if b := h.ReservedByte; !(b == ' ' || b == 0) {
		return false
	}

	if s := h.RecordStartTime; !(s.Hour <= 23 && s.Minute <= 59 && s.Second <= 60) {
		return false
	}

	return true
}

//TODO: needs v1.15
/*
// Hash can be used for deduping record blocks
func (h RecordHeader) Hash() string {
	var hm maphash.Hash

	hm.Write(h.NetworkIdentifier[:])
	hm.Write(h.StationIdentifier[:])
	hm.Write(h.LocationIdentifier[:])
	hm.Write(h.ChannelIdentifier[:])
	hm.Write([]byte{h.DataQualityIndicator})
	hm.Write(EncodeBTime(h.RecordStartTime))

	var parts [6]byte
	binary.BigEndian.PutUint16(parts[0:2], h.NumberOfSamples)
	binary.BigEndian.PutUint16(parts[2:4], uint16(h.SampleRateFactor))
	binary.BigEndian.PutUint16(parts[4:6], uint16(h.SampleRateMultiplier))
	hm.Write(parts[:])

	return fmt.Sprintf("%#x", hm.Sum64())
}
*/

// Less can be used for sorting record blocks.
func (h RecordHeader) Less(hdr RecordHeader) bool {
	switch {
	case h.Network() < hdr.Network():
		return true
	case h.Network() > hdr.Network():
		return false
	case h.Station() < hdr.Station():
		return true
	case h.Station() > hdr.Station():
		return false
	case h.Location() < hdr.Location():
		return true
	case h.Location() > hdr.Location():
		return false
	case h.Channel() < hdr.Channel():
		return true
	case h.Channel() > hdr.Channel():
		return false
	case h.DataQualityIndicator < hdr.DataQualityIndicator:
		return true
	case h.DataQualityIndicator > hdr.DataQualityIndicator:
		return false
	case h.StartTime().Before(hdr.StartTime()):
		return true
	case h.StartTime().After(hdr.StartTime()):
		return false
	case h.SampleRate() < hdr.SampleRate():
		return true
	default:
		return false
	}
}

// DecodeRecordHeader returns a RecordHeader from a byte slice.
func DecodeRecordHeader(data []byte) RecordHeader {
	var h [RecordHeaderSize]byte

	copy(h[:], data)

	return RecordHeader{
		SequenceNumber: func() [6]byte {
			var b [6]byte
			copy(b[:], h[0:6])
			return b
		}(),

		DataQualityIndicator: h[6],
		ReservedByte:         h[7],

		StationIdentifier: func() [5]byte {
			var b [5]byte
			copy(b[:], h[8:13])
			return b
		}(),
		LocationIdentifier: func() [2]byte {
			var b [2]byte
			copy(b[:], h[13:15])
			return b
		}(),
		ChannelIdentifier: func() [3]byte {
			var b [3]byte
			copy(b[:], h[15:18])
			return b
		}(),
		NetworkIdentifier: func() [2]byte {
			var b [2]byte
			copy(b[:], h[18:20])
			return b
		}(),

		RecordStartTime:      DecodeBTime(h[20:30]),
		NumberOfSamples:      binary.BigEndian.Uint16(h[30:32]),
		SampleRateFactor:     int16(binary.BigEndian.Uint16(h[32:34])),
		SampleRateMultiplier: int16(binary.BigEndian.Uint16(h[34:36])),

		ActivityFlags:    h[36],
		IOAndClockFlags:  h[37],
		DataQualityFlags: h[38],

		NumberOfBlockettesThatFollow: h[39],

		TimeCorrection:  int32(binary.BigEndian.Uint32(h[40:44])),
		BeginningOfData: binary.BigEndian.Uint16(h[44:46]),
		FirstBlockette:  binary.BigEndian.Uint16(h[46:48]),
	}
}

// EncodeRecordHeader converts a RecordHeader into a byte slice.
func EncodeRecordHeader(hdr RecordHeader) []byte {
	var b [RecordHeaderSize]byte

	copy(b[0:6], hdr.SequenceNumber[:])
	b[6] = hdr.DataQualityIndicator
	b[7] = hdr.ReservedByte

	copy(b[8:13], hdr.StationIdentifier[:])
	copy(b[13:15], hdr.LocationIdentifier[:])
	copy(b[15:18], hdr.ChannelIdentifier[:])
	copy(b[18:20], hdr.NetworkIdentifier[:])

	copy(b[20:30], EncodeBTime(hdr.RecordStartTime))
	binary.BigEndian.PutUint16(b[30:32], hdr.NumberOfSamples)
	binary.BigEndian.PutUint16(b[32:34], uint16(hdr.SampleRateFactor))
	binary.BigEndian.PutUint16(b[34:36], uint16(hdr.SampleRateMultiplier))

	b[36] = hdr.ActivityFlags
	b[37] = hdr.IOAndClockFlags
	b[38] = hdr.DataQualityFlags

	b[39] = hdr.NumberOfBlockettesThatFollow
	binary.BigEndian.PutUint32(b[40:44], uint32(hdr.TimeCorrection))
	binary.BigEndian.PutUint16(b[44:46], hdr.BeginningOfData)
	binary.BigEndian.PutUint16(b[46:48], hdr.FirstBlockette)

	h := make([]byte, RecordHeaderSize)
	copy(h[0:RecordHeaderSize], b[:])

	return h
}

func (h *RecordHeader) Unmarshal(data []byte) error {
	*h = DecodeRecordHeader(data)
	return nil
}

func (h RecordHeader) Marshal() ([]byte, error) {
	return EncodeRecordHeader(h), nil
}

// extact the sampling rate from the two seed factors.
func sampleRate(factor, multiplier int) float64 {
	var samprate float64

	switch f := float64(factor); {
	case factor > 0:
		samprate = f
	case factor < 0:
		samprate = -1.0 / f
	}

	switch m := float64(multiplier); {
	case multiplier > 0:
		samprate = samprate * m
	case multiplier < 0:
		samprate = -1.0 * (samprate / m)
	}

	return samprate
}

// flip the factors to get the sample interval
func samplePeriod(factor, multiplier int) time.Duration {
	if sps := sampleRate(factor, multiplier); sps > 0.0 {
		return time.Duration(float64(time.Second) / sps)
	}
	return 0
}

// Sets the bit at pos in the integer n.
func setBit(b byte, index uint8) byte {
	return b | (0x01 << index)
}

func clearBit(b byte, index uint8) byte {
	return b & (^(0x01 << index))
}

func isBitSet(b byte, index uint8) bool {
	return (b & (0x1 << index)) > 0
}
