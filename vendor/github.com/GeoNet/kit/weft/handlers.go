package weft

import (
	"bytes"
	"compress/gzip"
	"errors"
	"github.com/GeoNet/kit/metrics"
	"net/http"
	"reflect"
	"runtime"
	"strings"
	"sync"
)

var bufferPool = sync.Pool{
	New: func() interface{} {
		var b bytes.Buffer
		return &b
	},
}

var logger Logger = discarder{}

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

// RequestHandler should write the response for r into b and adjust h as required.
type RequestHandler func(r *http.Request, h http.Header, b *bytes.Buffer) error

// DirectRequestHandler allows writing to the http.ResponseWriter directly.
// Should return the number of bytes written to w and any errors.
type DirectRequestHandler func(r *http.Request, w http.ResponseWriter) (int64, error)

// ErrorHandler should write the error for err into b and adjust h as required.
// err can be nil
type ErrorHandler func(err error, h http.Header, b *bytes.Buffer) error

// MakeDirectHandler executes rh.  The caller should write directly to w for success (200) only.
// In the case of an rh returning an error ErrorHandler is executed and the response written to the client.
//
// Responses are counted.  rh is not wrapped with a timer as this includes the write to the client.
func MakeDirectHandler(rh DirectRequestHandler, eh ErrorHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		n, err := rh(r, w)
		if err == nil {
			metrics.StatusOK()
			metrics.Written(n)
			return
		}

		b := bufferPool.Get().(*bytes.Buffer)
		defer bufferPool.Put(b)
		b.Reset()

		setBestPracticeHeaders(w, r)

		e := eh(err, w.Header(), b)
		if e != nil {
			logger.Printf("setting error: %s", e.Error())
		}

		if w.Header().Get("Content-Type") == "" && b != nil {
			w.Header().Set("Content-Type", http.DetectContentType(b.Bytes()))
		}

		status := Status(err)

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
		metrics.Request()
		name := nameD(rh)

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
		}

	}
}

// MakeHandler returns an http.Handler that executes RequestHandler and collects timing information and metrics.
// In the case of errors ErrorHandler is used to set error content for the client.
// 50x errors are logged.
func MakeHandler(rh RequestHandler, eh ErrorHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b := bufferPool.Get().(*bytes.Buffer)
		defer bufferPool.Put(b)
		b.Reset()

		// run the RequestHandler with timing.  If this returns an error then use the
		// ErrorHandler to set the error content and header.

		t := metrics.Start()

		setBestPracticeHeaders(w, r)

		err := rh(r, w.Header(), b)

		if err != nil {
			e := eh(err, w.Header(), b)
			if e != nil {
				logger.Printf("setting error: %s", e.Error())
			}
		}

		t.Stop()

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
		name := name(rh)

		if e := t.Track(name + "." + r.Method); e != nil {
			logger.Printf("Track error: %s", e.Error())
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
		}
	}
}

/*
	These are recommended by Mozilla as part of the Observatory scan.
*/
func setBestPracticeHeaders(w http.ResponseWriter, r *http.Request) {
	//Content Security Policy: allow inline styles, but no inline scripts, prevent from clickjacking
	w.Header().Set("Content-Security-Policy", "default-src 'none'; "+
		"img-src 'self' *.geonet.org.nz data:; "+
		"font-src 'self' https://fonts.gstatic.com; "+
		"style-src 'self' 'unsafe-inline' https://*.googleapis.com; "+
		"script-src 'self' https://cdnjs.cloudflare.com https://www.google.com https://www.gstatic.com; "+
		"connect-src 'self' https://*.geonet.org.nz; "+
		"frame-src 'self' https://www.youtube.com https://www.google.com; "+
		"form-action 'self'; "+
		"base-uri 'none'; "+
		"frame-ancestors 'self'; "+
		"object-src 'self';")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Strict-Transport-Security", "max-age=63072000")
}

// TextError writes text errors to b for non nil error.
// Headers are set for intermediate caches.
//
// Implements ErrorHandler
func TextError(e error, h http.Header, b *bytes.Buffer) error {
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
func UseError(e error, h http.Header, b *bytes.Buffer) error {
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
func HTMLError(e error, h http.Header, b *bytes.Buffer) error {
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
func name(f RequestHandler) string {
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

// name finds the name of the function f
func nameD(f DirectRequestHandler) string {
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
