package ms

import (
	"bufio"
	"bytes"
	"fmt"
)

// Unpack decodes the record from a byte slice.
func (m *Record) Unpack(buf []byte) error {

	if len(buf) < RecordHeaderSize {
		return fmt.Errorf("unpack: given %v bytes; not enough to parse header", len(buf))
	}

	m.RecordHeader = DecodeRecordHeader(buf[0:RecordHeaderSize])
	if !m.RecordHeader.IsValid() {
		return fmt.Errorf("unpack: input is not a valid MSEED record: incorrect header")
	}

	pointer := m.RecordHeader.FirstBlockette //TODO: This could be replaced with bytes.Reader()
	for i := 0; i < int(m.RecordHeader.NumberOfBlockettesThatFollow); i++ {
		if pointer == 0 {
			return fmt.Errorf("unpack: next blockette pointer == 0 after %v blockettes", i-1)
		}

		//Get the blockette header
		if len(buf) < int(pointer+BlocketteHeaderSize) {
			return fmt.Errorf("unpack: given %v bytes; not enough to parse blockette header at %v", len(buf), pointer)
		}
		bhead := DecodeBlocketteHeader(buf[pointer : pointer+BlocketteHeaderSize])
		bpointer := pointer + BlocketteHeaderSize //start of blockette content

		switch bhead.BlocketteType {
		case 1000:
			if len(buf) < int(bpointer+Blockette1000Size) {
				return fmt.Errorf("unpack: given %v bytes; not enough to parse blockette 1000 at %v", len(buf), bpointer)
			}
			m.B1000 = DecodeBlockette1000(buf[bpointer : bpointer+Blockette1000Size])
		case 1001:
			if len(buf) < int(bpointer+Blockette1001Size) {
				return fmt.Errorf("unpack: given %v bytes; not enough to parse blockette 1001 at %v", len(buf), bpointer)
			}
			m.B1001 = DecodeBlockette1001(buf[bpointer : bpointer+Blockette1001Size])
		}

		pointer = bhead.NextBlockette
	}

	m.Data = make([]byte, len(buf)-int(m.RecordHeader.BeginningOfData))
	copy(m.Data, buf[m.RecordHeader.BeginningOfData:])

	return nil
}

func (m Record) Bytes() ([]byte, error) {
	switch enc := Encoding(m.B1000.Encoding); enc {
	case EncodingASCII:
		return trimRight(m.Data[:m.RecordHeader.NumberOfSamples]), nil
	default:
		return nil, fmt.Errorf("invalid encoding %v", enc)
	}
}

func (m Record) Strings() ([]string, error) {
	switch enc := Encoding(m.B1000.Encoding); enc {
	case EncodingASCII:
		var lines []string
		buf := bytes.NewBuffer(trimRight(m.Data[:m.RecordHeader.NumberOfSamples]))
		scanner := bufio.NewScanner(buf)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			return nil, err
		}
		return lines, nil
	default:
		return nil, fmt.Errorf("invalid encoding %v", enc)
	}
}

func (m Record) Int32s() ([]int32, error) {
	switch enc := Encoding(m.B1000.Encoding); enc {
	case EncodingInt32:
		return decodeInt32(m.Data, m.B1000.WordOrder, m.RecordHeader.NumberOfSamples)
	case EncodingSTEIM1:
		framecount := uint8(len(m.Data) / 64)
		if m.B1001.FrameCount != 0 {
			framecount = m.B1001.FrameCount
		}
		if int(framecount)*64 > len(m.Data) { //make sure the decoding doesn't overrun the buffer
			return nil, fmt.Errorf("unpack: header reported more bytes then are present in data packet: %v > %v", framecount*64, len(m.Data))
		}
		return decodeSteim(1, m.Data, m.B1000.WordOrder, framecount, m.RecordHeader.NumberOfSamples)
	case EncodingSTEIM2: //STEIM2
		framecount := uint8(len(m.Data) / 64)
		if m.B1001.FrameCount != 0 {
			framecount = m.B1001.FrameCount
		}
		if int(framecount)*64 > len(m.Data) { //make sure the decoding doesn't overrun the buffer
			return nil, fmt.Errorf("unpack: header reported more bytes then are present in data packet: %v > %v", framecount*64, len(m.Data))
		}
		return decodeSteim(2, m.Data, m.B1000.WordOrder, framecount, m.RecordHeader.NumberOfSamples)
	case EncodingIEEEFloat, EncodingIEEEDouble:
		samples, err := m.Float64s()
		if err != nil {
			return nil, err
		}
		var res []int32
		for _, v := range samples {
			res = append(res, int32(v))
		}
		return res, nil
	default:
		return nil, fmt.Errorf("invalid encoding %v", enc)
	}
}

func (m Record) Float64s() ([]float64, error) {
	switch enc := Encoding(m.B1000.Encoding); enc {
	case EncodingInt32, EncodingSTEIM1, EncodingSTEIM2:
		samples, err := m.Int32s()
		if err != nil {
			return nil, err
		}
		var res []float64
		for _, v := range samples {
			res = append(res, float64(v))
		}
		return res, nil
	case EncodingIEEEFloat:
		samples, err := decodeFloat32(m.Data, m.B1000.WordOrder, m.RecordHeader.NumberOfSamples)
		if err != nil {
			return nil, err
		}
		var res []float64
		for _, v := range samples {
			res = append(res, float64(v))
		}
		return res, nil
	case EncodingIEEEDouble:
		return decodeFloat64(m.Data, m.B1000.WordOrder, m.RecordHeader.NumberOfSamples)
	default:
		return nil, fmt.Errorf("invalid encoding %v", enc)
	}
}

func trimRight(data []byte) []byte {
	return bytes.TrimRight(data, "\x00")
}
