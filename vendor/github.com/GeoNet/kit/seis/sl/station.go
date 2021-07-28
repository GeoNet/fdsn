package sl

import (
	"encoding/binary"
	"strconv"
	"strings"
	"time"
)

// Station stores the latest state information for the given network and station combination.
type Station struct {
	Network   string    `json:"network"`
	Station   string    `json:"station"`
	Sequence  int       `json:"sequence"`
	Timestamp time.Time `json:"timestamp"`
}

// Key returns a blank Station except for the Network and Station entries, this useful as a map key.
func (s Station) Key() Station {
	return Station{
		Network: s.Network,
		Station: s.Station,
	}
}

// UnpackStation builds a Station based on a raw miniseed block header.
func UnpackStation(seq string, data []byte) Station {
	// miniseed heeader
	var header [50]byte
	copy(header[:], data)

	// sequence number
	var seqno int
	if no, err := strconv.ParseInt(strings.TrimSpace(seq), 16, 32); err == nil {
		seqno = int(no)
	}

	// network code
	var network [2]byte
	copy(network[:], header[18:20])

	// station code
	var station [5]byte
	copy(station[:], header[8:13])

	// start time
	var btime [10]byte
	copy(btime[:], header[20:30])

	// convert from miniseed time construct
	at := time.Date(
		int(binary.BigEndian.Uint16(btime[0:2])),
		1,
		1,
		int(btime[4]),
		int(btime[5]),
		int(btime[6]),
		int(binary.BigEndian.Uint16(btime[8:10]))*100000,
		time.UTC,
	).AddDate(0, 0, int(binary.BigEndian.Uint16(btime[2:4]))-1)

	// possible time correction
	var tc time.Duration
	if (header[36] & (0x1 << 1)) == 0 {
		tc = time.Duration(binary.BigEndian.Uint32(header[40:44]))
	}

	// TODO: don't ignore blockette time adjustments

	// pull it together
	return Station{
		Network:   strings.TrimSpace(string(network[:])),
		Station:   strings.TrimSpace(string(station[:])),
		Sequence:  seqno,
		Timestamp: at.Add(100 * time.Microsecond * tc),
	}
}
