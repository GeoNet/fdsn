package wefttest

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// client is used to make requests to the test server.
var client = &http.Client{Timeout: time.Second * 60}

// Request is for making requests to the server being tested.
// It describes the Request parameters and elements of the expected response.
type Request struct {
	ID             string // An identifier for the request.  Used in error messages.
	Method         string // Method for the request e.g., "PUT".  Defaults to "GET".
	Accept         string // Accept header for the request.  Defaults to */*
	URL            string // The URL to be tested e.g., /path/to/test.  The server can be added at test time.
	User, Password string // Credentials for basic auth if required.
	Status         int    // The expected HTTP status code for the request.  Defaults to http.StatusOK (200).
	Content        string // The expected content type.  Not tested if zero.  A zero Content-Type in the response is an error.
	Surrogate      string // The expected Surrogate-Control.  Not tested if zero.
}

type Requests []Request

// L returns the line of code it was called from.
func L() (loc string) {
	_, _, l, _ := runtime.Caller(1)
	return "L" + strconv.Itoa(l)
}

// DoAll runs all Request in Requests against server and returns for the first non nil error.
func (r Requests) DoAll(server string) error {
	for _, v := range r {
		_, err := v.Do(server)
		if err != nil {
			return err
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

	if res, err = client.Do(req); err != nil {
		return nil, fmt.Errorf("%s %s %s error: %s", r.ID, r.URL, r.Method, err.Error())
	}
	defer res.Body.Close()

	if r.Status != res.StatusCode {
		return nil, fmt.Errorf("%s %s %s got status %d expected %d", r.ID, r.URL, r.Method, res.StatusCode, r.Status)
	}

	if r.Surrogate != "" {
		if res.Header.Get("Surrogate-Control") != r.Surrogate {
			return nil, fmt.Errorf("%s got Surrogate-Control %s expected %s", r.ID, res.Header.Get("Surrogate-Control"), r.Surrogate)
		}
	}

	if r.Content != "" {
		if res.Header.Get("Content-Type") != r.Content {
			return nil, fmt.Errorf("%s %s %s got Content-Type %s expected %s", r.ID, r.URL, r.Method, res.Header.Get("Content-Type"), r.Content)
		}
	}

	if res.Header.Get("Content-Type") == "" {
		return nil, fmt.Errorf("%s got empty Content-Type header for response", r.ID)
	}

	return ioutil.ReadAll(res.Body)
}
