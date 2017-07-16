// Package msg is for processing messages.
package msg

import (
	"github.com/GeoNet/fdsn/internal/platform/metrics"
)

// Message defines an interface for message processing.
type Processor interface {
	Process([]byte) error
}

// DoProcess executes m.Process with metrics.
func DoProcess(m Processor, b []byte) error {
	metrics.MsgRx()
	t := metrics.Start()
	defer t.Track("process")

	s := m.Process(b)

	switch s {
	case nil:
		metrics.MsgProc()
	default:
		metrics.MsgErr()
	}

	return s
}
