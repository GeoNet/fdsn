package fdsn

import (
	"bytes"
	"fmt"
	"strings"
	"time"
)

const WsMarshalTimeFormat = "2006-01-02T15:04:05"

var EmptyWsDateTime = WsDateTime{time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)}
var ZeroWsDateTime = WsDateTime{time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)}

type WsDateTime struct {
	time.Time
}

func (t *WsDateTime) UnmarshalText(text []byte) error {
	return _unmarshalWsTime(text, &t.Time)
}

func (t WsDateTime) MarshalText() ([]byte, error) {
	return []byte(t.Time.UTC().Format(WsMarshalTimeFormat)), nil
}

func _unmarshalWsTime(text []byte, t *time.Time) (err error) {
	// we allows query parameter with or without timezone 'Z'
	// (in our code we only dealt with 'Z' so official go package can handle it easily)
	// note: not accepting timezones other than 'Z', as FDSN only using UTC
	s := string(bytes.TrimSpace(text))
	if len(s) == 10 {
		*t, err = time.Parse(time.DateOnly, s)
		return err
	}
	if !strings.HasSuffix(s, "Z") {
		s += "Z"
	}
	if *t, err = time.Parse(time.RFC3339Nano, s); err != nil {
		return fmt.Errorf("invalid time string: %s - %s", s, err.Error())
	}
	return nil
}
