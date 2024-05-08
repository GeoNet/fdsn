package wefttest

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// client is used to make requests to the test server.
var client = &http.Client{Timeout: time.Second * 60}

var httpMethods = []string{"GET", "DELETE", "POST", "PUT", "HEAD", "CONNECT", "OPTIONS", "TRACE", "PATCH"}

var noncePattern = "^'nonce-[A-Za-z0-9+/=]{20}' 'strict-dynamic' %s$"

// Request is for making requests to the server being tested.
// It describes the Request parameters and elements of the expected response.
type Request struct {
	ID             string            // An identifier for the request.  Used in error messages.
	Method         string            // Method for the request e.g., "PUT".  Defaults to "GET".
	PostBody       []byte            // Request body.
	Accept         string            // Accept header for the request.  Defaults to */*
	URL            string            // The URL to be tested e.g., /path/to/test.  The server can be added at test time.
	User, Password string            // Credentials for basic auth if required.
	Status         int               // The expected HTTP status code for the request.  Defaults to http.StatusOK (200).
	Content        string            // The expected content type.  Not tested if zero.  A zero Content-Type in the response is an error.
	Surrogate      string            // The expected Surrogate-Control.  Not tested if zero.
	CSP            map[string]string // expected header for content-security-policy
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

	if r.PostBody != nil {
		req, err = http.NewRequest(r.Method, r.URL, bytes.NewReader(r.PostBody))
	} else {
		req, err = http.NewRequest(r.Method, r.URL, nil)
	}

	if err != nil {
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

	if r.CSP != nil { //check response csp
		cspString := res.Header.Get(("content-security-policy"))
		if err := checkCSP(parseCspContent(cspString), r.CSP); err != nil {
			//fmt.Println("## error URL", r.URL, cspString)
			return nil, fmt.Errorf("%s got error content-security-policy header url %s, \n error: %s", r.ID, r.URL, err.Error())
		}
	}

	return io.ReadAll(res.Body)
}

/**
 * convert CSP string to map
 */
func parseCspContent(content string) map[string]string {
	cspArray := strings.Split(content, ";")
	cspMap := make(map[string]string)
	for _, csp := range cspArray {
		s := strings.TrimSpace(csp)
		index := strings.Index(s, " ")
		if index > 0 {
			key := s[0:index]
			val := s[index+1:]
			cspMap[key] = strings.TrimSpace(val)
		}
	}
	return cspMap
}

/**
 * check response CSP match expected
 */
func checkCSP(respCsp, expectedCsp map[string]string) error {
	l1 := len(respCsp)
	l2 := len(expectedCsp)
	if l1 != l2 {
		return fmt.Errorf("## Response CSP count %v doesn't match expected %v", l1, l2)
	}
	for k, v := range expectedCsp {
		v1 := respCsp[k]
		if k == "script-src" && strings.Contains(v1, "nonce-") { //check nonce
			escapedV := strings.Replace(v, "*", "\\*", -1) // escape wildcards to avoid regex clash
			pattern := fmt.Sprintf(noncePattern, escapedV)
			if !patternMatch(pattern, v1) {
				return fmt.Errorf("## Response CSP %s=%s doesn't match expected %s=%s", k, v1, k, v)
			}
		} else {
			if v1 != v {
				return fmt.Errorf("## Response CSP %s=%s doesn't match expected %s=%s", k, v1, k, v)
			}
		}
	}
	return nil
}

func patternMatch(pattern string, str string) bool {
	match, err := regexp.MatchString(pattern, str)
	if err != nil {
		return false
	}
	return match
}

// MethodNotAllowed tests r that all HTTP methods that are not in allowed return an http.StatusMethodNotAllowed.
// r.Surrogate and r.Content should be set for an http.StatusMethodNotAllowed response before calling.
func (r Request) MethodNotAllowed(server string, allowed []string) (int, error) {
	r.Status = http.StatusMethodNotAllowed

	i := 0

METHOD:
	for _, v := range httpMethods {
		for _, a := range allowed {
			if a == v {
				continue METHOD
			}
		}

		r.Method = v

		_, err := r.Do(server)
		if err != nil {
			return 0, err
		}
		i++
	}

	return i, nil
}

// ExtraParameter adds key and value to r.URL and checks that this causes an http.StatusBadRequest.
// r.Surrogate and r.Content should be set for an http.StatusBadRequest response before calling.
func (r Request) ExtraParameter(server, key, value string) error {
	if strings.HasPrefix(r.URL, "/") {
		r.URL = server + r.URL
	}

	req, err := http.NewRequest(r.Method, r.URL, nil)
	if err != nil {
		return err
	}

	switch len(req.URL.Query()) {
	case 0:
		r.URL = r.URL + "?" + key + "=" + value
	default:
		r.URL = r.URL + "&" + key + "=" + value
	}

	r.Status = http.StatusBadRequest

	_, err = r.Do(server)

	return err
}

// FuzzPath tests r if there is a path only (no query parameters) using fuzz.
// The fuzzed path can return a http.StatusNotFound or a http.StatusBadRequest.
// r.Surrogate and r.Content should be unset before calling (unless they are the same for http.StatusNotFound and http.StatusBadRequest).
// Some fuzzed URL paths may not parse as a valid URL and these are skipped.
func (r Request) FuzzPath(server string, fuzz []string) (int, error) {
	if strings.HasPrefix(r.URL, "/") {
		r.URL = server + r.URL
	}

	req, err := http.NewRequest(r.Method, r.URL, nil)
	if err != nil {
		return 0, err
	}

	// there are query parameters - return
	if len(req.URL.Query()) != 0 {
		return 0, nil
	}

	i := 0

	// fuzzed routes should 404 or possibly 400.
	// most common should be 404, test that first, if it fails check for 400
	for _, f := range fuzz {
		r.Status = http.StatusNotFound
		r.URL = req.URL.Path + f

		// skip any fuzzed URLs that can't be parsed as a URL.
		_, err := http.NewRequest(r.Method, server+r.URL, nil)
		if err != nil {
			continue
		}

		_, err = r.Do(server)
		if err != nil {
			r.Status = http.StatusBadRequest
			_, err = r.Do(server)
			if err != nil {
				return 0, fmt.Errorf("%s for URL %s", err.Error(), r.URL)
			}
		}

		i++
	}

	return i, nil
}

// FuzzQuery tests the URL query parameter (if there are any) using fuzz and checks that this returns http.StatusBadRequest.
// Query parameters are changes one at a time for all values in fuzz.
// r.Surrogate and r.Content should be set for an http.StatusBadRequest response before calling.
func (r Request) FuzzQuery(server string, fuzz []string) (int, error) {
	if strings.HasPrefix(r.URL, "/") {
		r.URL = server + r.URL
	}

	req, err := http.NewRequest(r.Method, r.URL, nil)
	if err != nil {
		return 0, err
	}

	// no query parameters to fuzz. return
	// fuzz the URL and check all 404 then return.
	if len(req.URL.Query()) == 0 {
		return 0, nil
	}

	i := 0
	r.Status = http.StatusNotFound

	r.Status = http.StatusBadRequest

	for _, f := range fuzz {
		for k := range req.URL.Query() {
			tmp := req.URL.Query()
			tmp.Set(k, f)
			r.URL = req.URL.Path + "?" + tmp.Encode()

			_, err = r.Do(server)
			if err != nil {
				return 0, fmt.Errorf("%s for URL %s", err.Error(), r.URL)
			}

			i++
		}
	}

	return i, nil
}
