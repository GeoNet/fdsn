package weft

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"runtime"
	"strings"
	"sync"

	"github.com/GeoNet/kit/metrics"
)

var bufferPool = sync.Pool{
	New: func() interface{} {
		var b bytes.Buffer
		return &b
	},
}

var logger Logger = discarder{}
var logPostBody = false
var logReq = false

const (
	GZIP = "gzip"
)

// Compressible types from https://www.fastly.com/blog/new-gzip-settings-and-deciding-what-compress
var compressibleMimes = map[string]bool{
	"text/html":                     true,
	"text/html; charset=utf-8":      true,
	"application/x-javascript":      true,
	"text/css":                      true,
	"application/javascript":        true,
	"text/javascript":               true,
	"text/plain":                    true,
	"text/xml":                      true,
	"application/json":              true,
	"application/vnd.ms-fontobject": true,
	"application/x-font-opentype":   true,
	"application/x-font-truetype":   true,
	"application/x-font-ttf":        true,
	"application/xml":               true,
	"font/eot":                      true,
	"font/opentype":                 true,
	"font/otf":                      true,
	"image/svg+xml":                 true,
	"image/vnd.microsoft.icon":      true,
	// other types
	"application/vnd.geo+json": true,
	"application/cap+xml":      true,
	"text/csv":                 true,
}

var defaultCsp = map[string]string{
	"default-src":     "'none'",
	"img-src":         "'self' *.geonet.org.nz data: https://*.google-analytics.com https://*.googletagmanager.com",
	"font-src":        "'self' https://fonts.gstatic.com",
	"style-src":       "'self'",
	"script-src":      "'self'",
	"connect-src":     "'self' https://*.geonet.org.nz https://*.google-analytics.com https://*.analytics.google.com https://*.googletagmanager.com",
	"frame-src":       "'self' https://www.youtube.com https://www.google.com",
	"form-action":     "'self' https://*.geonet.org.nz",
	"base-uri":        "'none'",
	"frame-ancestors": "'self'",
	"object-src":      "'none'",
}

/**
 * RequestHandler should write the response for r into b and adjust h as required
 */
type RequestHandler func(r *http.Request, h http.Header, b *bytes.Buffer) error

/**
 * RequestHandlerWithNonce for pages with strict scripts csp
 * @param nonce: string to be passed to page template as attribute of scripts
 * refer https://csp.withgoogle.com/docs/strict-csp.html
 */
type RequestHandlerWithNonce func(r *http.Request, h http.Header, b *bytes.Buffer, nonce string) error

// DirectRequestHandler allows writing to the http.ResponseWriter directly.
// Should return the number of bytes written to w and any errors.
type DirectRequestHandler func(r *http.Request, w http.ResponseWriter) (int64, error)

// ErrorHandler should write the error for err into b and adjust h as required.
// err can be nil
type ErrorHandler func(err error, h http.Header, b *bytes.Buffer, nonce string) error

// MakeDirectHandler executes rh.  The caller should write directly to w for success (200) only.
// In the case of an rh returning an error ErrorHandler is executed and the response written to the client.
//
// Responses are counted.  rh is not wrapped with a timer as this includes the write to the client.
func MakeDirectHandler(rh DirectRequestHandler, eh ErrorHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b := bufferPool.Get().(*bytes.Buffer)
		defer bufferPool.Put(b)

		name := name(rh)

		b.Reset()

		// run the RequestHandler with timing.  If this returns an error then use the
		// ErrorHandler to set the error content and header.
		t := metrics.Start()
		// note: the ending `writeResponseAndLogMetrics` calls t.Track which will stop the metric timer, too
		//set csp headers
		SetBestPracticeHeaders(w, r, nil, "")
		//run request handler
		n, err := rh(r, w)
		if err == nil { //all good, return
			metrics.StatusOK()
			metrics.Request()
			metrics.Written(n)

			if er := t.Track(name + "." + r.Method); er != nil {
				logger.Printf("error tracking metric : %s", er.Error())
			}

			return
		}

		//everything below are for error responses
		logRequest(r)
		t.Stop()

		//run error handler
		e := eh(err, w.Header(), b, "")
		if e != nil {
			logger.Printf("setting error: %s", e.Error())
		}
		//write error response and log metrics
		writeResponseAndLogMetrics(err, w, r, b, name, nil)
	}
}

// MakeHandler with default CSP policy
func MakeHandler(rh RequestHandler, eh ErrorHandler) http.HandlerFunc {
	return MakeHandlerWithCsp(rh, eh, nil)
}

// MakeHandler with specified CSP policy
// MakeHandler returns an http.Handler that executes RequestHandler and collects timing information and metrics.
// In the case of errors ErrorHandler is used to set error content for the client.
// 50x errors are logged.
func MakeHandlerWithCsp(rh RequestHandler, eh ErrorHandler, customCsp map[string]string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b := bufferPool.Get().(*bytes.Buffer)
		defer bufferPool.Put(b)
		b.Reset()

		// run the RequestHandler with timing.  If this returns an error then use the
		// ErrorHandler to set the error content and header.
		t := metrics.Start()
		//run request handler
		err := rh(r, w.Header(), b)
		if err != nil {
			//run error handler
			e := eh(err, w.Header(), b, "")
			if e != nil {
				logger.Printf("2 error from error handler: %s", e.Error())
			}
			SetBestPracticeHeaders(w, r, defaultCsp, "")
		} else {
			SetBestPracticeHeaders(w, r, customCsp, "")
		}

		logRequest(r)

		t.Stop()

		//write error response and log metrics
		name := name(rh)
		writeResponseAndLogMetrics(err, w, r, b, name, &t)
	}
}

/*
 * MakeHandler with default CSP policy and RequestHandlerWithNonce
 * a randomly generated nonce is passed to RequestHandlerWithNonce
 */
func MakeHandlerWithNonce(rh RequestHandlerWithNonce, eh ErrorHandler) http.HandlerFunc {
	return MakeHandlerWithCspNonce(rh, eh, nil)
}

// MakeHandler with specified CSP policy and RequestHandlerWithNonce which accept a nonce string
// to be used in page template (which need nonce for scripts)
// returns an http.Handler that executes RequestHandler and collects timing information and metrics.
// In the case of errors ErrorHandler is used to set error content for the client.
// 50x errors are logged.
func MakeHandlerWithCspNonce(rh RequestHandlerWithNonce, eh ErrorHandler, customCsp map[string]string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b := bufferPool.Get().(*bytes.Buffer)
		defer bufferPool.Put(b)
		b.Reset()

		// run the RequestHandler with timing.  If this returns an error then use the
		// ErrorHandler to set the error content and header.
		t := metrics.Start()
		//get a random nonce string
		nonce, err := getCspNonce(16)
		if err == nil {
			//run request handler
			err = rh(r, w.Header(), b, nonce)
		}
		if err != nil {
			//run error handler
			e := eh(err, w.Header(), b, nonce)
			if e != nil {
				logger.Printf("2 error from error handler: %s", e.Error())
			}
			SetBestPracticeHeaders(w, r, defaultCsp, nonce)
		} else {
			SetBestPracticeHeaders(w, r, customCsp, nonce)
		}

		logRequest(r)

		t.Stop()

		//write error response and log metrics
		name := name(rh)
		writeResponseAndLogMetrics(err, w, r, b, name, &t)
	}
}

/**
 * write http response and metrics logging
 */
func writeResponseAndLogMetrics(err error, w http.ResponseWriter, r *http.Request, b *bytes.Buffer, name string, t *metrics.Timer) {
	// serve the content (which could now be error content).  Gzipping if required.
	if w.Header().Get("Content-Type") == "" && b != nil {
		w.Header().Set("Content-Type", http.DetectContentType(b.Bytes()))
	}
	status := Status(err)
	var n int64
	// keep errors for writing to client separate from errors that came from the request handler.
	// log them but don't add metrics
	var writeErr error

	switch status {
	case http.StatusMovedPermanently:
		http.Redirect(w, r, err.Error(), http.StatusMovedPermanently)
		metrics.StatusOK()
	case http.StatusSeeOther:
		http.Redirect(w, r, err.Error(), http.StatusSeeOther)
		metrics.StatusOK()
	default:
		w.Header().Add("Vary", "Accept-Encoding")

		//remove trailing content-type information, like ';version=2'
		contentType := w.Header().Get("Content-Type")
		i := strings.Index(contentType, ";")
		if i > 0 {
			contentType = contentType[0:i]
		}
		contentType = strings.TrimSpace(contentType)

		if strings.Contains(r.Header.Get("Accept-Encoding"), GZIP) && compressibleMimes[contentType] && b.Len() > 20 {
			w.Header().Set("Content-Encoding", GZIP)
			gz := gzip.NewWriter(w)
			defer gz.Close()
			w.WriteHeader(status)
			n, writeErr = b.WriteTo(gz)
		} else {
			w.WriteHeader(status)
			n, writeErr = b.WriteTo(w)
		}
	}

	if writeErr != nil {
		logger.Printf("error writing to w: %s", writeErr.Error())
	}

	// request metrics and logging
	metrics.Written(n)
	if t != nil {
		if e := t.Track(name + "." + r.Method); e != nil {
			logger.Printf("Track error: %s", e.Error())
		}
	}

	metrics.Request()

	switch status {
	case http.StatusOK, http.StatusMovedPermanently, http.StatusSeeOther, http.StatusGone, http.StatusNoContent:
		metrics.StatusOK()
	case http.StatusBadRequest:
		metrics.StatusBadRequest()
		logger.Printf("%d %s", status, r.RequestURI)
	case http.StatusUnauthorized:
		metrics.StatusUnauthorized()
		logger.Printf("%d %s", status, r.RequestURI)
	case http.StatusNotFound:
		metrics.StatusNotFound()
		logger.Printf("%d %s", status, r.RequestURI)
	case http.StatusInternalServerError:
		metrics.StatusInternalServerError()
		logger.Printf("%d %s %s %s %s", status, r.Method, r.RequestURI, name, err.Error())
	case http.StatusServiceUnavailable:
		metrics.StatusServiceUnavailable()
		logger.Printf("%d %s %s %s %s", status, r.Method, r.RequestURI, name, err.Error())
	case http.StatusTooManyRequests:
		metrics.StatusTooManyRequests()
		logger.Printf("%d %s %s %s %s", status, r.Method, r.RequestURI, name, err.Error())
	}
}

/*
 * These are recommended by Mozilla as part of the Observatory scan.
 * NOTE: customCsp should include the whole set of an item as it override that in defaultCsp
 * @param nonce: string to be added to script CSP, refer: https://csp.withgoogle.com/docs/strict-csp.html
 */
func SetBestPracticeHeaders(w http.ResponseWriter, r *http.Request, customCsp map[string]string, nonce string) {
	var csp strings.Builder
	for k, v := range defaultCsp {
		s := v
		if customCsp != nil {
			if v1, ok := customCsp[k]; ok {
				s = v1
			}
		}
		csp.WriteString(k)
		csp.WriteString(" ")

		if k == "script-src" && nonce != "" && s != "'none'" { //add nonce to CSP
			csp.WriteString(fmt.Sprintf(" 'nonce-%s' 'strict-dynamic' ", nonce))
		}
		csp.WriteString(s)
		csp.WriteString("; ")
	}
	w.Header().Set("Content-Security-Policy", csp.String())
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Strict-Transport-Security", "max-age=63072000")
	w.Header().Set("Referrer-Policy", "no-referrer")
}

func logRequest(r *http.Request) {
	if logReq {
		logger.Printf("%s - %s - %s\n", r.RemoteAddr, r.Method, r.RequestURI)
	}
	if logPostBody {
		switch r.Method {
		case http.MethodPost, http.MethodPut:
			body, err := io.ReadAll(r.Body)
			if err != nil {
				logger.Printf("Error reading request body") // This error doesn't affect we processing requests
			} else {
				logger.Printf("Body:%s", string(body))
				// put read bytes back so the real handler can use it
				r.Body = io.NopCloser(bytes.NewBuffer(body))
			}
		}
	}
}

// TextError writes text errors to b for non nil error.
// Headers are set for intermediate caches.
//
// Implements ErrorHandler
func TextError(e error, h http.Header, b *bytes.Buffer, nonce string) error {
	if b == nil {
		return errors.New("nil *bytes.Buffer")
	}

	var err error

	switch Status(e) {
	case http.StatusOK:
		return nil
	case http.StatusNoContent:
		return nil
	case http.StatusMovedPermanently:
		h.Set("Surrogate-Control", "max-age=86400")
		return nil
	case http.StatusSeeOther:
		h.Set("Surrogate-Control", "max-age=86400")
		return nil
	case http.StatusGone:
		h.Set("Surrogate-Control", "max-age=86400")
		_, err = b.WriteString("this resource no longer exists")
		h.Set("Content-Type", "text/plain; charset=utf-8")
	case http.StatusNotFound:
		b.Reset()
		h.Set("Surrogate-Control", "max-age=10")
		h.Set("Content-Type", "text/plain; charset=utf-8")
		_, err = b.WriteString("not found")
	case http.StatusBadRequest:
		b.Reset()
		h.Set("Surrogate-Control", "max-age=86400")
		h.Set("Content-Type", "text/plain; charset=utf-8")
		_, err = b.WriteString("bad request: " + e.Error())
	case http.StatusMethodNotAllowed:
		b.Reset()
		h.Set("Surrogate-Control", "max-age=86400")
		h.Set("Content-Type", "text/plain; charset=utf-8")
		_, err = b.WriteString("method not allowed")
	case http.StatusServiceUnavailable, http.StatusInternalServerError:
		b.Reset()
		h.Set("Surrogate-Control", "max-age=10")
		h.Set("Content-Type", "text/plain; charset=utf-8")
		_, err = b.WriteString("service unavailable please try again soon")
	}

	return err
}

// UseError sets Headers are set for intermediate caches.
// The content of b is not changed.
//
// Implements ErrorHandler
func UseError(e error, h http.Header, b *bytes.Buffer, nonce string) error {
	switch Status(e) {
	case http.StatusOK:
	case http.StatusNoContent:
	case http.StatusMovedPermanently:
		h.Set("Surrogate-Control", "max-age=86400")
	case http.StatusSeeOther:
		h.Set("Surrogate-Control", "max-age=86400")
	case http.StatusGone:
		h.Set("Surrogate-Control", "max-age=86400")
	case http.StatusNotFound:
		b.Reset()
		h.Set("Surrogate-Control", "max-age=10")
	case http.StatusBadRequest:
		b.Reset()
		h.Set("Surrogate-Control", "max-age=86400")
	case http.StatusMethodNotAllowed:
		b.Reset()
		h.Set("Surrogate-Control", "max-age=86400")
	case http.StatusServiceUnavailable, http.StatusInternalServerError:
		b.Reset()
		h.Set("Surrogate-Control", "max-age=10")
	}

	return nil
}

// HTMLError writes error pages to b for non nil error.
// Headers are set for intermediate caches.
//
// Implements ErrorHandler
func HTMLError(e error, h http.Header, b *bytes.Buffer, nonce string) error {
	if b == nil {
		return errors.New("nil *bytes.Buffer")
	}

	var err error

	switch Status(e) {
	case http.StatusOK:
		return nil
	case http.StatusNoContent:
		return nil
	case http.StatusMovedPermanently:
		h.Set("Surrogate-Control", "max-age=86400")
		return nil
	case http.StatusSeeOther:
		h.Set("Surrogate-Control", "max-age=86400")
		return nil
	case http.StatusGone:
		h.Set("Surrogate-Control", "max-age=86400")
		_, err = b.Write([]byte(ErrGone))
		h.Set("Content-Type", "text/html; charset=utf-8")
	case http.StatusNotFound:
		b.Reset()
		h.Set("Surrogate-Control", "max-age=10")
		_, err = b.Write([]byte(ErrNotFound))
		h.Set("Content-Type", "text/html; charset=utf-8")
	case http.StatusBadRequest:
		b.Reset()
		h.Set("Surrogate-Control", "max-age=86400")
		_, err = b.Write([]byte(ErrBadRequest))
		h.Set("Content-Type", "text/html; charset=utf-8")
	case http.StatusMethodNotAllowed:
		b.Reset()
		h.Set("Surrogate-Control", "max-age=86400")
		_, err = b.Write([]byte(ErrMethodNotAllowed))
	case http.StatusServiceUnavailable, http.StatusInternalServerError:
		b.Reset()
		h.Set("Surrogate-Control", "max-age=10")
		_, err = b.Write([]byte(ErrServiceUnavailable))
		h.Set("Content-Type", "text/html; charset=utf-8")
	}

	return err
}

// name finds the name of the function f
func name(f interface{}) string {
	var n string
	// Find the name of the function f to use as the timer id
	fn := runtime.FuncForPC(reflect.ValueOf(f).Pointer())
	if fn != nil {
		n = fn.Name()
		i := strings.LastIndex(n, ".")
		if i > 0 && i+1 < len(n) {
			n = n[i+1:]
		}
	}
	return n
}

// NoMatch returns a 404 for GET requests.
//
// Implements RequestHandler
func NoMatch(r *http.Request, h http.Header, b *bytes.Buffer) error {
	err := CheckQuery(r, []string{"GET"}, []string{}, []string{})
	if err != nil {
		return err
	}

	return StatusError{Code: http.StatusNotFound}
}

// Up returns a 200 and simple page for GET requests.
//
// Implements RequestHandler
func Up(r *http.Request, h http.Header, b *bytes.Buffer) error {
	err := CheckQuery(r, []string{"GET"}, []string{}, []string{})
	if err != nil {
		return err
	}

	h.Set("Content-Type", "text/html; charset=utf-8")

	b.Write([]byte("<html><head></head><body>up</body></html>"))

	return nil
}

// Soh returns a 200 and simple page for GET requests.
//
// Implements RequestHandler
func Soh(r *http.Request, h http.Header, b *bytes.Buffer) error {
	err := CheckQuery(r, []string{"GET"}, []string{}, []string{})
	if err != nil {
		return err
	}

	h.Set("Content-Type", "text/html; charset=utf-8")

	b.Write([]byte("<html><head></head><body>ok</body></html>"))

	return nil
}

// ReturnDefaultCSP returns the default Content Security Policy used
// by handlers. This is a copy of the map, so can be changed safely if needed.
func ReturnDefaultCSP() map[string]string {
	copy := make(map[string]string)
	for k, v := range defaultCsp {
		copy[k] = v
	}
	return copy
}
