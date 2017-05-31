// holdings is for retrieving data holding information from miniSEED files.
package holdings

import (
	"github.com/GeoNet/kit/mseed"
	"io"
	"strings"
	"time"
)

// the record length of the miniSEED records.  Constant for all GNS miniSEED files.
const recordLength int = 512

type Holding struct {
	Network, Station, Channel, Location string
	Start                               time.Time
	NumSamples                          int
}

// SingleStream reads miniSEED from r in 512 byte records and returns a summary.
// Expects a single stream (not multiplexed miniSEED) in r.
func SingleStream(r io.Reader) (Holding, error) {
	msr := mseed.NewMSRecord()
	defer mseed.FreeMSRecord(msr)

	record := make([]byte, recordLength)

	// read the first record and use it to set up h.
	// a non nil error can be the end of the Reader (EOF),
	// a short record or some other error.
	_, err := io.ReadFull(r, record)
	switch {
	case err == io.EOF:
		return Holding{}, nil
	case err != nil:
		return Holding{}, err
	}

	err = msr.Unpack(record, recordLength, 1, 0)
	if err != nil {
		return Holding{}, err
	}

	h := Holding{
		Network:    strings.Trim(msr.Network(), "\x00"),
		Station:    strings.Trim(msr.Station(), "\x00"),
		Channel:    strings.Trim(msr.Channel(), "\x00"),
		Location:   strings.Trim(msr.Location(), "\x00"),
		Start:      msr.Starttime(),
		NumSamples: int(msr.Numsamples()),
	}

loop:
	for {
		_, err = io.ReadFull(r, record)
		switch {
		case err == io.EOF:
			break loop
		case err != nil:
			return Holding{}, err
		}

		err = msr.Unpack(record, recordLength, 1, 0)
		if err != nil {
			return Holding{}, err
		}

		h.NumSamples += int(msr.Numsamples())
	}

	return h, nil
}
