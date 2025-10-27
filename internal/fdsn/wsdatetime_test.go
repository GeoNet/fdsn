package fdsn_test

import (
	"testing"
	"time"

	"github.com/GeoNet/fdsn/internal/fdsn"
)

func TestTimeParse(t *testing.T) {
	var tm fdsn.WsDateTime

	if err := tm.UnmarshalText([]byte("2015-01-12T12:12:12.999999")); err != nil {
		t.Error(err)
	}

	if err := tm.UnmarshalText([]byte("2015-01-12T12:12:12")); err != nil {
		t.Error(err)
	}

	if err := tm.UnmarshalText([]byte("2015-01-12")); err != nil {
		t.Error(err)
	}

	if err := tm.UnmarshalText([]byte("2015-01-12T12:12:12-09:00")); err == nil {
		t.Error("expected an error for invalid time string.")
	}
}

// confirm that we'll trim subseconds
func TestMarshalWsDateTime(t *testing.T) {
	tests := []struct {
		time     time.Time
		expected string
	}{
		{
			time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC),
			"2025-12-31T23:59:59",
		},
		{
			time.Date(2025, 12, 31, 23, 59, 59, 123, time.UTC),
			"2025-12-31T23:59:59",
		},
		{
			time.Date(2025, 12, 31, 23, 59, 59, 123456, time.UTC),
			"2025-12-31T23:59:59",
		},
	}

	for _, v := range tests {
		tm := fdsn.WsDateTime{v.time}
		if s, _ := tm.MarshalText(); string(s) != v.expected {
			t.Errorf("expected %s got %s", v.expected, string(s))
		}

	}
}
