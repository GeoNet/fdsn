package mtrapp

import (
	"github.com/GeoNet/mtr/internal"
	"sync/atomic"
)

// Increment these counters as required.
var (
	Requests                  = Counter{id: internal.Requests}                  // HTTP requests
	StatusOK                  = Counter{id: internal.StatusOK}                  // HTTP status 200
	StatusBadRequest          = Counter{id: internal.StatusBadRequest}          // HTTP status 400
	StatusUnauthorized        = Counter{id: internal.StatusUnauthorized}        // HTTP status 401
	StatusNotFound            = Counter{id: internal.StatusNotFound}            // HTTP status 404
	StatusInternalServerError = Counter{id: internal.StatusInternalServerError} // HTTP status 500
	StatusServiceUnavailable  = Counter{id: internal.StatusServiceUnavailable}  // HTTP status 503
	MsgRx                     = Counter{id: internal.MsgRx}                     // Message Received.
	MsgTx                     = Counter{id: internal.MsgTx}                     // Message transmitted.
	MsgProc                   = Counter{id: internal.MsgProc}                   // Message processed.
	MsgErr                    = Counter{id: internal.MsgErr}                    // Message error.
)

var counters = [...]*Counter{
	&Requests,
	&StatusOK,
	&StatusBadRequest,
	&StatusUnauthorized,
	&StatusNotFound,
	&StatusInternalServerError,
	&StatusServiceUnavailable,
	&MsgRx,
	&MsgTx,
	&MsgProc,
	&MsgErr,
}

var lastVal [len(counters)]uint64
var currVal [len(counters)]uint64

// Counter is for counting events.  It is safe for concurrent access.
type Counter struct {
	i  uint64
	id internal.ID
}

// Inc increments the counter by 1.
func (c *Counter) Inc() {
	atomic.AddUint64(&c.i, 1)
}

func (c *Counter) value() uint64 {
	return atomic.LoadUint64(&c.i)
}
