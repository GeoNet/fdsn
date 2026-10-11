package main

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/GeoNet/kit/weft"
)

var mux *http.ServeMux

func initRoutes() {
	mux = http.NewServeMux()

	mux.HandleFunc("/", weft.MakeHandler(weft.NoMatch, weft.TextError))
	mux.HandleFunc("/soh/up", weft.MakeHandler(weft.Up, weft.TextError))
	mux.HandleFunc("/soh", weft.MakeHandler(soh, weft.UseError))

	mux.HandleFunc("/fdsnws/dataselect/1/", weft.MakeHandler(fdsnDataselectV1Index, weft.TextError))
	mux.HandleFunc("/fdsnws/dataselect/1/query", weft.MakeDirectHandler(fdsnDataselectV1Handler, fdsnErrorHandler))
	mux.HandleFunc("/fdsnws/dataselect/1/version", weft.MakeHandler(fdsnDataselectVersion, weft.TextError))
	mux.HandleFunc("/fdsnws/dataselect/1/application.wadl", weft.MakeHandler(fdsnDataselectWadl, weft.TextError))
}

func soh(r *http.Request, h http.Header, b *bytes.Buffer) error {
	err := weft.CheckQuery(r, []string{"GET"}, []string{}, []string{})
	if err != nil {
		return err
	}
	b.WriteString("<html><head></head><body>ok</body></html>")
	return nil
}

const FDSN_ERR_FORMAT = `Error %03d: %s
%s
Usage details are available from https://www.geonet.org.nz/data/tools/FDSN
Request:
%s
Request Submitted:
%s
Service version:
%s`

type fdsnError struct {
	weft.StatusError
	url       string
	timestamp time.Time
}

func fdsnErrorHandler(err error, h http.Header, b *bytes.Buffer, nounce string) error {
	switch e := err.(type) {
	case fdsnError:
		h.Set("Content-Type", "text/plain; charset=utf-8")
		if e.Code != http.StatusNoContent && e.Code != http.StatusNotFound {
			var ver string
			if strings.HasPrefix(e.url, "/fdsnws/dataselect/") {
				ver = dataselectVersion
			}
			msg := fmt.Sprintf(FDSN_ERR_FORMAT, e.Code, http.StatusText(e.Code), e.Err, e.url, e.timestamp.Format(time.RFC3339), ver)
			b.WriteString(msg)
		}
		return nil
	}

	return weft.TextError(err, h, b, nounce)
}
