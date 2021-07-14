package ms

import (
	"encoding/binary"
	"fmt"
	"math"
)

func decodeInt32(data []byte, order uint8, samples uint16) ([]int32, error) {
	if n := uint16(len(data) / 4); n < samples {
		return nil, fmt.Errorf("invalid data length: %d", n)
	}

	var values []int32
	for i := 0; i < int(samples*4); i += 4 {
		var b uint32
		switch order {
		case 0:
			b = binary.LittleEndian.Uint32(data[i:])
		default:
			b = binary.BigEndian.Uint32(data[i:])
		}
		values = append(values, int32(b))
	}

	return values, nil
}

func decodeFloat32(data []byte, order uint8, samples uint16) ([]float32, error) {
	if n := uint16(len(data) / 4); n < samples {
		return nil, fmt.Errorf("invalid data length: %d", n)
	}
	var values []float32
	for i := 0; i < int(samples*4); i += 4 {
		var b uint32
		switch order {
		case 0:
			b = binary.LittleEndian.Uint32(data[i:])
		default:
			b = binary.BigEndian.Uint32(data[i:])
		}
		values = append(values, math.Float32frombits(b))
	}
	return values, nil
}

func decodeFloat64(data []byte, order uint8, samples uint16) ([]float64, error) {
	if n := uint16(len(data) / 8); n < samples {
		return nil, fmt.Errorf("invalid data length: %d", n)
	}
	var values []float64
	for i := 0; i < int(samples*8); i += 8 {
		var b uint64
		switch order {
		case 0:
			b = binary.LittleEndian.Uint64(data[i:])
		default:
			b = binary.BigEndian.Uint64(data[i:])
		}
		values = append(values, math.Float64frombits(b))
	}
	return values, nil
}
