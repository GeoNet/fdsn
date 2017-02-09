package wefttest

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"runtime"
	"strconv"
	"strings"
)

// Client is used to make requests to the test server.
var Client = &http.Client{}

// Request is for making requests to the server being tested.
// It describes the Request parameters and elements of the expected response.
type Request struct {
	// An identifier for the request.  Used in error messages.
	ID string
	// Method for the request e.g., "PUT".  Defaults to "GET".
	Method string
	// Accept header for the request.  Defaults to */*
	Accept string
	// The URL to be tested e.g.., /path/to/test.  The server can be added at test time.
	URL string
	// Credentials for basic auth if required.
	User, Password string
	// The expected HTTP status code for the request.  Defaults to http.StatusOK (200)
	Status int
	// The expected content type.  Not tested if zero.
	Content string
	// The expected Surrogate-Control.  Not tested if zero.
	Surrogate string
}

type Requests []Request

// L returns the line of code it was called from.
func L() (loc string) {
	_, _, l, _ := runtime.Caller(1)
	return "L" + strconv.Itoa(l)
}

// DoAllStatusOK runs all Requests in r that should return http.StatusOK (200).
// Returns for the first non nil error.
func (r Requests) DoAllStatusOk(server string) error {
	for _, v := range r {
		if v.Status == 0 || v.Status == http.StatusOK {
			if _, err := v.Do(server); err != nil {
				return err
			}
		}
	}
	return nil
}

// DoCheckQuery runs the requests with additional query parameters and cache
// busters.  Returns for the first non nil error.
// It's an error if any of the requests don't return http.StatusBadRequest.
// Content-Type type is expected to be for error pages and Surrogate-Control
// should be the max allowable.
func (r Requests) DoCheckQuery(server string) error {
	// Check with a query param cache buster.  This is the same as
	// extra query parameters.
	for _, v := range r {
		if v.Status == 0 || v.Status == http.StatusOK {
			v.Status = http.StatusBadRequest

			if strings.Contains(v.URL, "?") {
				v.URL = v.URL + "&cache=busta"
			} else {
				v.URL = v.URL + "?cache=busta"
			}

			if _, err := v.Do(server); err != nil {
				return fmt.Errorf("%s busted fail: %s", v.ID, err.Error())
			}
		}
	}

	for _, v := range r {
		if v.Status == 0 || v.Status == http.StatusOK {
			v.Status = http.StatusBadRequest

			if !strings.Contains(v.URL, "?") {
				v.URL = v.URL + ";cache=busta"
				if _, err := v.Do(server); err != nil {
					return fmt.Errorf("%s busted fail: %s", v.ID, err.Error())
				}
			}
		}
	}

	return nil
}

// Do runs the Request and returns a non nil error for responses that
// do not match the Request (and other error conditions).
// server is prepended to r.URL if it starts with "/".
// Also returns the body from the response.
func (r Request) Do(server string) ([]byte, error) {
	// Set default values for request
	if r.Accept == "" {
		r.Accept = "*/*"
	}

	if r.Status == 0 {
		r.Status = http.StatusOK
	}

	if r.Method == "" {
		r.Method = "GET"
	}

	if strings.HasPrefix(r.URL, "/") {
		r.URL = server + r.URL
	}

	var req *http.Request
	var res *http.Response
	var err error

	if req, err = http.NewRequest(r.Method, r.URL, nil); err != nil {
		return nil, err
	}

	req.Header.Add("Accept", r.Accept)

	if r.User != "" || r.Password != "" {
		req.SetBasicAuth(r.User, r.Password)
	}

	if res, err = Client.Do(req); err != nil {
		return nil, fmt.Errorf("%s %s %s error: %s", r.ID, r.URL, r.Method, err.Error())
	}
	defer res.Body.Close()

	if r.Status != res.StatusCode {
		return nil, fmt.Errorf("%s %s %s got status %d expected %d", r.ID, r.URL, r.Method, res.StatusCode, r.Status)
	}

	switch res.StatusCode {
	// bad requests should return a max surrogate cache time and an error page or message.
	case http.StatusBadRequest:
		if res.Header.Get("Surrogate-Control") != "max-age=86400" {
			return nil, fmt.Errorf("%s %s got Surrogate-Control %s expected max-age=86400",
				r.URL, r.Method, res.Header.Get("Surrogate-Control"))
		}
		switch res.Header.Get("Content-Type") {
		case "text/html; charset=utf-8", "text/plain; charset=utf-8":
		default:
			return nil, fmt.Errorf("%s %s %s got Content-Type %s expected text/html; charset=utf-8 or text/html; charset=utf-8",
				r.ID, r.URL, r.Method, res.Header.Get("Content-Type"))
		}
	default:
		// Content-Type should not be empty
		if res.Header.Get("Content-Type") == "" {
			return nil, fmt.Errorf("%s %s %s got empty Content-Type.", r.ID, r.URL, r.Method)
		}

		// Surrogate-Control for intermediate caches.
		if r.Surrogate != "" {
			if res.Header.Get("Surrogate-Control") != r.Surrogate {
				return nil, fmt.Errorf("%s %s %s got Surrogate-Control %s expected %s",
					r.ID, r.URL, r.Method, res.Header.Get("Surrogate-Control"), r.Surrogate)
			}
		}

		// If the request expected a Content-Type it should match the response.
		if r.Content != "" {
			if res.Header.Get("Content-Type") != r.Content {
				return nil, fmt.Errorf("%s %s %s got Content-Type %s expected %s",
					r.ID, r.URL, r.Method, res.Header.Get("Content-Type"), r.Content)
			}
		}
	}

	return ioutil.ReadAll(res.Body)
}
