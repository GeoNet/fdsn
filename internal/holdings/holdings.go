// holdings is for retrieving data holding information from miniSEED files.
package holdings

import (
	"io"
	"time"

	ms "github.com/GeoNet/kit/seis/ms"
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

	msr, err := ms.NewRecord(record)
	if err != nil {
		return Holding{}, err
	}

	h := Holding{
		Network:    msr.Network(),
		Station:    msr.Station(),
		Channel:    msr.Channel(),
		Location:   msr.Location(),
		Start:      msr.StartTime(),
		NumSamples: int(msr.NumberOfSamples),
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

		msr, err = ms.NewRecord(record)
		if err != nil {
			return Holding{}, err
		}

		h.NumSamples += int(msr.NumberOfSamples)
	}

	return h, nil
}
