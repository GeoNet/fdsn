package valid

import (
	"fmt"
	"net/http"
	"regexp"
	"time"
)

var (
	pid, pidErr = regexp.Compile(`^[0-9]+[a-z]?[0-9]+$`) // quake public ids are of the form 2013p407387 or a number e.g., 345679
)

type Validator func(string) error

// implements weft.Error
type Error struct {
	Code int
	Err  error
}

func (s Error) Error() string {
	if s.Err == nil {
		return "<nil>"
	}
	return s.Err.Error()
}

func (s Error) Status() int {
	return s.Code
}

// PublicID for validating quake publicIDs
func PublicID(s string) error {
	if pidErr != nil {
		return pidErr
	}

	if pid.MatchString(s) {
		return nil
	}

	return Error{Code: http.StatusBadRequest, Err: fmt.Errorf("invalid publicID: %s", s)}
}

func ParseDate(s string) (time.Time, error) {
	if s == "" {
		return time.Now().UTC(), nil
	}

	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, Error{Code: http.StatusBadRequest, Err: fmt.Errorf("invalid date: %s", s)}
	}

	return d, nil
}
