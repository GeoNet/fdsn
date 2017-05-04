package msg

import (
	"log"
	"time"
)

var hearbeatAge = time.Duration(-5) * time.Minute

type HeartBeat struct {
	ServiceID string
	SentTime  time.Time
	err       error
}

// Logs receipt of h.
// Returns true if the h is old.  False if not.
func (h *HeartBeat) RxLog() bool {
	if h.err != nil {
		return true
	}

	b := h.SentTime.Before(time.Now().UTC().Add(hearbeatAge))
	switch b {
	case true:
		log.Printf("Received old heartbeat for %s", h.ServiceID)
	case false:
		log.Printf("Received heartbeat for %s", h.ServiceID)
	}

	return b
}

func (h *HeartBeat) TxLog() {
	if h.err != nil {
		return
	}

	log.Printf("Sending heartbeat for %s", h.ServiceID)
}

func (h *HeartBeat) Err() error {
	return h.err
}

func (h *HeartBeat) SetErr(err error) {
	h.err = err
}
