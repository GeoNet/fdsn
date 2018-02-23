package valid_test

import (
	"github.com/GeoNet/fdsn/internal/valid"
	"net/http"
	"runtime"
	"strconv"
	"testing"
)

var bad = &valid.Error{Code: http.StatusBadRequest}

func TestPublicID(t *testing.T) {
	in := []struct {
		s   string
		fn  valid.Validator
		err *valid.Error
		id  string
	}{
		{s: "2013p407387", fn: valid.PublicID, id: loc()},
		{s: "1407387", fn: valid.PublicID, id: loc()},
		{s: "2013pp407387", fn: valid.PublicID, err: bad, id: loc()},
		{s: "2013pp407387", fn: valid.PublicID, err: bad, id: loc()},
	}

	for _, v := range in {
		err := v.fn(v.s)

		checkError(t, v.id, v.err, err)
	}
}

func checkError(t *testing.T, id string, expected *valid.Error, actual error) {
	if actual != nil {
		if expected == nil {
			t.Errorf("%s nil expected error with non nil actual error", id)
			return
		}
	}

	if expected == nil {
		return
	}

	if actual == nil {
		t.Errorf("%s nil actual error for non nil expected error", id)
		return
	}

	switch a := actual.(type) {
	case valid.Error:
		if a.Code != expected.Code {
			t.Errorf("%s expected code %d got %d", id, expected.Code, a.Code)
		}
	default:
		t.Errorf("%s actual error is not of type Error", id)
	}
}

func loc() string {
	_, _, l, _ := runtime.Caller(1)
	return "L" + strconv.Itoa(l)
}
