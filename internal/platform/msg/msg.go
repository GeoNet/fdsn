// Package msg is for processing messages.
package msg

import (
	"github.com/GeoNet/mtr/mtrapp"
)

// Message defines an interface for message processing.
type Processor interface {
	Process([]byte) error
}

// DoProcess executes m.Process with metrics.
func DoProcess(m Processor, b []byte) error {
	mtrapp.MsgRx.Inc()
	t := mtrapp.Start()
	defer t.Track("process")

	s := m.Process(b)

	switch s {
	case nil:
		mtrapp.MsgProc.Inc()
	default:
		mtrapp.MsgErr.Inc()
	}

	return s
}
