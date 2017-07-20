// pkg metrics is for gathering metrics.
package metrics

import (
	"sync/atomic"
	"time"
)

var msgCounters [4]uint64
var msgLast [4]uint64
var msgCurrent [4]uint64

var httpCounters [7]uint64
var httpLast [7]uint64
var httpCurrent [7]uint64

// A MsgCounters records message counters.
type MsgCounters struct {
	// Rx is the count of messages received.
	Rx uint64

	// Tx is the count of messages transmitted.
	Tx uint64

	// Proc is the count of messages processed.
	Proc uint64

	// Err is the count of errored messages.
	Err uint64

	// At is the time the counters were sampled at.
	At time.Time
}

// A HttpCounters records http counters.
type HttpCounters struct {
	// Request is the count of http requests.
	Request uint64

	// StatusOK is the count of http 200 responses.
	StatusOK uint64

	// StatusBadRequest is the count of http 400 responses.
	StatusBadRequest uint64

	// StatusUnauthorized is the count of http 401 responses.
	StatusUnauthorized uint64

	// StatusNotFound is the count of http 404 responses.
	StatusNotFound uint64

	// StatusInternalServerError is the count of http 500 responses.
	StatusInternalServerError uint64

	// StatusServiceUnavailable is the count of http 503 responses.
	StatusServiceUnavailable uint64

	// At is the time the counters were sampled at.
	At time.Time
}

// ReadMsgCounters populates m with message counter delta values
// since last time it was called.
func ReadMsgCounters(m *MsgCounters) {
	m.At = time.Now().UTC()

	for i := range msgCounters {
		msgCurrent[i] = atomic.LoadUint64(&msgCounters[i])
	}

	m.Rx = msgCurrent[0] - msgLast[0]
	m.Tx = msgCurrent[1] - msgLast[1]
	m.Proc = msgCurrent[2] - msgLast[2]
	m.Err = msgCurrent[3] - msgLast[3]

	for i := range msgCounters {
		msgLast[i] = msgCurrent[i]
	}
}

// ReadHttpCounters populates m with http counter delta values
// since last time it was called.
func ReadHttpCounters(m *HttpCounters) {
	m.At = time.Now().UTC()

	for i := range httpCounters {
		httpCurrent[i] = atomic.LoadUint64(&httpCounters[i])
	}

	m.Request = httpCurrent[0] - httpLast[0]
	m.StatusOK = httpCurrent[1] - httpLast[1]
	m.StatusBadRequest = httpCurrent[2] - httpLast[2]
	m.StatusUnauthorized = httpCurrent[3] - httpLast[3]
	m.StatusNotFound = httpCurrent[4] - httpLast[4]
	m.StatusInternalServerError = httpCurrent[5] - httpLast[5]
	m.StatusServiceUnavailable = httpCurrent[6] - httpLast[6]

	for i := range httpCounters {
		httpLast[i] = httpCurrent[i]
	}
}

// MsgRx increments the message received counter. It is safe for concurrent access.
func MsgRx() {
	atomic.AddUint64(&msgCounters[0], 1)
}

// MsgTx increments the message transmitted counter. It is safe for concurrent access.
func MsgTx() {
	atomic.AddUint64(&msgCounters[1], 1)
}

// MsgProc increments the message processed counter. It is safe for concurrent access.
func MsgProc() {
	atomic.AddUint64(&msgCounters[2], 1)
}

// MsgErr increments the message error counter. It is safe for concurrent access.
func MsgErr() {
	atomic.AddUint64(&msgCounters[3], 1)
}

// Request increments the http request counter. It is safe for concurrent access.
func Request() {
	atomic.AddUint64(&httpCounters[0], 1)
}

// StatusOK increments the http response 200 counter. It is safe for concurrent access.
func StatusOK() {
	atomic.AddUint64(&httpCounters[1], 1)
}

// StatusBadRequest increments the http response 400 counter. It is safe for concurrent access.
func StatusBadRequest() {
	atomic.AddUint64(&httpCounters[2], 1)
}

// StatusUnauthorized increments the http response 400 counter. It is safe for concurrent access.
func StatusUnauthorized() {
	atomic.AddUint64(&httpCounters[3], 1)
}

// StatusNotFound increments the http response 404 counter. It is safe for concurrent access.
func StatusNotFound() {
	atomic.AddUint64(&httpCounters[4], 1)
}

// StatusInternalServerError increments the http response 500 counter. It is safe for concurrent access.
func StatusInternalServerError() {
	atomic.AddUint64(&httpCounters[5], 1)
}

// StatusServiceUnavailable increments the http response 503 counter. It is safe for concurrent access.
func StatusServiceUnavailable() {
	atomic.AddUint64(&httpCounters[6], 1)
}
