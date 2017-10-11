package weft

import (
	"bytes"
	"compress/gzip"
	"log"
	"net/http"
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

var compressibleMimes = map[string]bool{
	// Compressible types from https://www.fastly.com/blog/new-gzip-settings-and-deciding-what-compress
	"text/html":                     true,
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

var surrogateControl = map[int]string{
	http.StatusNotFound:            "max-age=10",
	http.StatusServiceUnavailable:  "max-age=10",
	http.StatusInternalServerError: "max-age=10",
	http.StatusBadRequest:          "max-age=86400",
	http.StatusMethodNotAllowed:    "max-age=86400",
	http.StatusMovedPermanently:    "max-age=86400",
}

/*
MakeHandler executes f and writes the response in b to the client
with gzipping and Surrogate-Control headers.

HTML error pages are written to the client when res.Code is not http.StatusOK.
*/
func MakeHandlerPage(f RequestHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t := metrics.Start()

		b := bufferPool.Get().(*bytes.Buffer)
		defer bufferPool.Put(b)
		b.Reset()

		res := f(r, w.Header(), b)
		t.Stop()

		switch res.Code {
		case http.StatusMovedPermanently:
			w.Header().Set("Surrogate-Control", surrogateControl[http.StatusMovedPermanently])
			http.Redirect(w, r, res.Redirect, http.StatusMovedPermanently)
		case http.StatusSeeOther:
			http.Redirect(w, r, res.Redirect, http.StatusSeeOther)

			// change the Code to 200 for adding to the metrics.
			// 303 is a successful post followed by a GET redirect.
			res.Code = http.StatusOK
		default:
			WriteBytes(w, r, res, b, true)
		}

		t.Track(name(f) + "." + r.Method)
		res.Count()

		res.log(r)

		if t.Taken() > 250 {
			log.Printf("slow: took %d ms serving %s", t.Taken(), r.RequestURI)
		}
	}
}

/*
MakeHandlerAPI executes f.

When res.Code is not http.StatusOK the contents of res.Msg are written to w.

Surrogate-Control headers are also set for intermediate caches.
*/
func MakeHandlerAPI(f RequestHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t := metrics.Start()
		var res *Result

		b := bufferPool.Get().(*bytes.Buffer)
		defer bufferPool.Put(b)
		b.Reset()

		res = f(r, w.Header(), b)
		t.Stop()
		WriteBytes(w, r, res, b, false)

		t.Track(name(f) + "." + r.Method)
		res.Count()

		res.log(r)

		if t.Taken() > 250 {
			log.Printf("slow: took %d ms serving %s", t.Taken(), r.RequestURI)
		}
	}
}

/*
MakeSimpleHandler executes f.  The caller should write directly to w for success (200) only.
When res.Code is not http.StatusOK the contents of res.Msg are written to w.

Responses are counted.  f is not wrapped with a timer as this includes the write to the client.
*/
func MakeSimpleHandler(f SimpleRequestHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var res *Result

		res = f(r, w)

		// if we have StatusOK it means we've already written this to the header, so only handle other cases
		if res.Code != http.StatusOK {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			Write(w, r, res)
		}

		res.Count()
		res.log(r)
	}
}

/*
WriteBytes writes the contents of b to w.  Appropriate response headers are set.
The response is gzipped if appropriate for the client and the content.
Surrogate-Control headers are also set for intermediate caches.
Surrogate-Control set calling WriteBytes will be respected for res.Code == http.StatusOK
and overwritten for other Codes.

In the case of res.Code being for an error then HTML error pages or res.Msg is written
to w depending on errorPage.

If b is nil then only headers are written to w.
*/
func WriteBytes(w http.ResponseWriter, r *http.Request, res *Result, b *bytes.Buffer, errorPage bool) {
	if res.Code == 0 {
		res.Code = http.StatusOK
		log.Printf("WARN: weft - received Result.Code == 0, serving 200.")
	}

	if w.Header().Get("Surrogate-Control") == "" {
		w.Header().Set("Surrogate-Control", "max-age=10")
	}

	if res.Code != 200 {
		switch errorPage {
		case true:
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			if b != nil {
				b.Reset()
				if e, ok := errorPages[res.Code]; ok {
					b.Write(e)
				} else {
					b.Write(errorPages[http.StatusInternalServerError])
				}
			}
		case false:
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			if b != nil {
				b.Reset()
				b.WriteString(res.Msg)
			}
		}

		if s, ok := surrogateControl[res.Code]; ok {
			w.Header().Set("Surrogate-Control", s)
		} else {
			w.Header().Set("Surrogate-Control", "max-age=10")
		}
	}

	/*
	 write the response.  With gzipping if possible.
	*/

	w.Header().Add("Vary", "Accept-Encoding")

	if w.Header().Get("Content-Type") == "" && b != nil {
		w.Header().Set("Content-Type", http.DetectContentType(b.Bytes()))
	}

	if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") && b != nil && b.Len() > 20 {

		contentType := w.Header().Get("Content-Type")

		i := strings.Index(contentType, ";")
		if i > 0 {
			contentType = contentType[0:i]
		}

		contentType = strings.TrimSpace(contentType)

		if compressibleMimes[contentType] {
			w.Header().Set("Content-Encoding", "gzip")
			gz := gzip.NewWriter(w)
			defer gz.Close()
			w.WriteHeader(res.Code)
			b.WriteTo(gz)

			return
		}
	}

	w.WriteHeader(res.Code)
	if b != nil {
		b.WriteTo(w)
	}
}

/*
Write writes a header response to the client and in the case of
res.Code != http.StatusOK also writes res.Msg.

Surrogate-Control headers are also set for intermediate caches.
Surrogate-Control set calling Write will be respected for
res.Code == http.StatusOK and overwritten for other Codes.
*/
func Write(w http.ResponseWriter, r *http.Request, res *Result) {
	if res.Code == 0 {
		res.Code = http.StatusOK
		log.Printf("WARN: weft - received Result.Code == 0, serving 200.")
	}

	switch res.Code {
	case http.StatusOK:
		if w.Header().Get("Surrogate-Control") == "" {
			w.Header().Set("Surrogate-Control", "max-age=10")
		}

		w.WriteHeader(res.Code)
	default:
		if s, ok := surrogateControl[res.Code]; ok {
			w.Header().Set("Surrogate-Control", s)
		} else {
			w.Header().Set("Surrogate-Control", "max-age=10")
		}

		w.WriteHeader(res.Code)
		w.Write([]byte(res.Msg))
	}
}
