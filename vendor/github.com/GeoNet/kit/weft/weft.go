// weft helps with web applications.
package weft

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/GeoNet/kit/metrics"
)

// QueryValidator returns an error for any invalid query parameter values.
type QueryValidator func(url.Values) error

// Logger defines an interface for logging.
type Logger interface {
	Printf(string, ...interface{})
}

type discarder struct {
}

func (d discarder) Printf(string, ...interface{}) {
}

// Error represents an error with an associated HTTP status code.
type Error interface {
	error
	Status() int // the HTTP status code for the error.
}

// StatusError is for errors with HTTP status codes.  If Code is http.StatusBadRequest then Err.Error()
// should return a message that is suitable for returning to the client.
// If Code is http.StatusMovedPermanently or http.StatusSeeOther then Err.Error should return the redirect URL.
type StatusError struct {
	Code int
	Err  error
}

// SetLogger sets the logger used for logging.  If not set log messages are discarded.
func SetLogger(l Logger) {
	if l != nil {
		logger = l
	}
}

// EnableLogRequest makes logger to log all requests
func EnableLogRequest(b bool) {
	logReq = b
}

// EnableLogPostBody makes logger to log post body
func EnableLogPostBody(b bool) {
	logPostBody = b
}

// DataDog initialises sending metrics to DataDog.
func DataDog(apiKey, hostName, appName string, logger Logger) {
	metrics.DataDogHttp(apiKey, hostName, appName, logger)
}

func (s StatusError) Error() string {
	if s.Err == nil {
		return "<nil>"
	}
	return s.Err.Error()
}

func (s StatusError) Status() int {
	return s.Code
}

// Status returns the HTTP status code appropriate for err.
// It returns:
//   - http.StatusOk if err is nil
//   - err.Code if err is a StatusErr and Code is set
//   - otherwise http.StatusServiceUnavailable
func Status(err error) int {
	if err == nil {
		return http.StatusOK
	}

	switch e := err.(type) {
	case Error:
		switch e.Status() {
		case 0:
			return http.StatusServiceUnavailable
		default:
			return e.Status()
		}

	default:
		return http.StatusServiceUnavailable
	}
}

/*
CheckQuery inspects r and makes sure that the method is allowed, that all required query parameters
are present, and that no more than the required and optional parameters are present.
*/
func CheckQuery(r *http.Request, method, required, optional []string) error {
	var ok bool

	for i := range method {
		if r.Method == method[i] {
			ok = true
		}
	}

	if !ok {
		return StatusError{Code: http.StatusMethodNotAllowed, Err: errors.New("method not allowed")}
	}

	if strings.Contains(r.URL.RawQuery, ";") {
		return StatusError{Code: http.StatusBadRequest, Err: errors.New("found a cache buster")}
	}

	v := r.URL.Query()

	if len(required) == 0 && len(optional) == 0 {
		if len(v) == 0 {
			return nil
		} else {
			return StatusError{Code: http.StatusBadRequest, Err: errors.New("found unexpected query parameters")}
		}
	}

	var missing []string

	for _, k := range required {
		if v.Get(k) == "" {
			missing = append(missing, k)
		} else {
			v.Del(k)
		}
	}

	switch len(missing) {
	case 0:
	case 1:
		return StatusError{Code: http.StatusBadRequest, Err: errors.New("missing required query parameter: " + missing[0])}
	default:
		return StatusError{Code: http.StatusBadRequest, Err: errors.New("missing required query parameters: " + strings.Join(missing, ", "))}
	}

	for _, k := range optional {
		v.Del(k)
	}

	if len(v) > 0 {
		return StatusError{Code: http.StatusBadRequest, Err: errors.New("found additional query parameters")}
	}

	return nil
}

// CheckQueryValid calls CheckQuery and then validates the query parameters using f.
// It is an error for f to be nil.
func CheckQueryValid(r *http.Request, method, required, optional []string, f QueryValidator) (url.Values, error) {
	if f == nil {
		return nil, StatusError{Code: http.StatusInternalServerError, Err: errors.New("nil QueryValidator")}
	}

	err := CheckQuery(r, method, required, optional)
	if err != nil {
		return nil, err
	}

	q := r.URL.Query()

	err = f(q)
	if err != nil {
		return nil, err
	}

	return q, nil
}
