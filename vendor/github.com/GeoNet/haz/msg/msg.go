// Package msg provides interfaces and methods for processing messages.
//
// Models a message flow that looks like:
//   [transport] receive -> decode -> process -> encode -> send [transport]
package msg

import (
	"github.com/GeoNet/mtr/mtrapp"
	"log"
)

type Raw struct {
	Subject       string
	Body          string
	ReceiptHandle string
}

// Message defines an interface that allows for message processing.
type Message interface {
	Process() (reprocess bool) // a hint to try to reprocess the message.
	Err() error
}

// Process executes m.Process with logging and metrics.
func Process(m Message) bool {
	mtrapp.MsgRx.Inc()
	t := mtrapp.Start()

	s := m.Process()

	t.Track("process")
	mtrapp.MsgProc.Inc()

	if m.Err() != nil {
		log.Printf("%s", m.Err().Error())
		mtrapp.MsgErr.Inc()
	}

	return s
}
