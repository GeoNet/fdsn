package metrics_test

import (
	"github.com/GeoNet/fdsn/internal/platform/metrics"
	"runtime"
	"strconv"
	"testing"
)

func TestMsgCounters(t *testing.T) {
	testCases := []struct {
		i string
		f func()
		e metrics.MsgCounters
	}{
		{i: l(), f: metrics.MsgRx, e: metrics.MsgCounters{Rx: 1}},
		{i: l(), f: metrics.MsgTx, e: metrics.MsgCounters{Tx: 1}},
		{i: l(), f: metrics.MsgProc, e: metrics.MsgCounters{Proc: 1}},
		{i: l(), f: metrics.MsgErr, e: metrics.MsgCounters{Err: 1}},
	}

	var m metrics.MsgCounters

	for _, v := range testCases {
		// check all the counters are 0
		metrics.ReadMsgCounters(&m)

		if m.Rx != 0 {
			t.Errorf("msgRx expected 0 got %d", m.Rx)
		}

		if m.Tx != 0 {
			t.Errorf("msgTx expected 0 got %d", m.Tx)
		}

		if m.Proc != 0 {
			t.Errorf("msgProc expected 0 got %d", m.Proc)
		}

		if m.Err != 0 {
			t.Errorf("msgErr expected 0 got %d", m.Err)
		}

		// increment one counter
		// and check we incremented the correct counter
		v.f()

		metrics.ReadMsgCounters(&m)

		if m.Rx != v.e.Rx {
			t.Errorf("%s msgRx expected %d got %d", v.i, v.e.Rx, m.Rx)
		}

		if m.Tx != v.e.Tx {
			t.Errorf("%s msgTx expected %d got %d", v.i, v.e.Tx, m.Tx)
		}

		if m.Proc != v.e.Proc {
			t.Errorf("%s msgProc expected %d got %d", v.i, v.e.Proc, m.Proc)
		}

		if m.Err != v.e.Err {
			t.Errorf("%s msgErr expected %d got %d", v.i, v.e.Err, m.Err)
		}
	}
}

func TestHttpCounters(t *testing.T) {
	testCases := []struct {
		i string
		f func()
		e metrics.HttpCounters
	}{
		{i: l(), f: metrics.Request, e: metrics.HttpCounters{Request: 1}},
		{i: l(), f: metrics.StatusOK, e: metrics.HttpCounters{StatusOK: 1}},
		{i: l(), f: metrics.StatusBadRequest, e: metrics.HttpCounters{StatusBadRequest: 1}},
		{i: l(), f: metrics.StatusUnauthorized, e: metrics.HttpCounters{StatusUnauthorized: 1}},
		{i: l(), f: metrics.StatusNotFound, e: metrics.HttpCounters{StatusNotFound: 1}},
		{i: l(), f: metrics.StatusInternalServerError, e: metrics.HttpCounters{StatusInternalServerError: 1}},
		{i: l(), f: metrics.StatusServiceUnavailable, e: metrics.HttpCounters{StatusServiceUnavailable: 1}},
	}

	var m metrics.HttpCounters

	for _, v := range testCases {
		// check all the counters are 0
		metrics.ReadHttpCounters(&m)

		if m.Request != 0 {
			t.Errorf("expected 0 got %d", m.Request)
		}
		if m.StatusOK != 0 {
			t.Errorf("expected 0 got %d", m.StatusOK)
		}
		if m.StatusBadRequest != 0 {
			t.Errorf("expected 0 got %d", m.StatusBadRequest)
		}
		if m.StatusUnauthorized != 0 {
			t.Errorf("expected 0 got %d", m.StatusUnauthorized)
		}
		if m.StatusNotFound != 0 {
			t.Errorf("expected 0 got %d", m.StatusNotFound)
		}
		if m.StatusInternalServerError != 0 {
			t.Errorf("expected 0 got %d", m.StatusInternalServerError)
		}
		if m.StatusServiceUnavailable != 0 {
			t.Errorf("expected 0 got %d", m.StatusServiceUnavailable)
		}

		// increment one counter
		// and check we incremented the correct counter
		v.f()

		metrics.ReadHttpCounters(&m)

		if m.Request != v.e.Request {
			t.Errorf("%s Request expected %d got %d", v.i, v.e.Request, m.Request)
		}
		if m.StatusOK != v.e.StatusOK {
			t.Errorf("%s StatusOK expected %d got %d", v.i, v.e.StatusOK, m.StatusOK)
		}
		if m.StatusBadRequest != v.e.StatusBadRequest {
			t.Errorf("%s StatusBadRequest expected %d got %d", v.i, v.e.StatusBadRequest, m.StatusBadRequest)
		}
		if m.StatusUnauthorized != v.e.StatusUnauthorized {
			t.Errorf("%s StatusUnauthorized expected %d got %d", v.i, v.e.StatusUnauthorized, m.StatusUnauthorized)
		}
		if m.StatusNotFound != v.e.StatusNotFound {
			t.Errorf("%s StatusNotFound expected %d got %d", v.i, v.e.StatusNotFound, m.StatusNotFound)
		}
		if m.StatusInternalServerError != v.e.StatusInternalServerError {
			t.Errorf("%s StatusInternalServerError expected %d got %d", v.i, v.e.StatusInternalServerError, m.StatusInternalServerError)
		}
		if m.StatusServiceUnavailable != v.e.StatusServiceUnavailable {
			t.Errorf("%s StatusServiceUnavailable expected %d got %d", v.i, v.e.StatusServiceUnavailable, m.StatusServiceUnavailable)
		}

	}
}

// l returns the line of code it was called from.
func l() (loc string) {
	_, _, l, _ := runtime.Caller(1)
	return "L" + strconv.Itoa(l)
}
