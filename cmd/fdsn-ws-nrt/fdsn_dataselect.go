package main

import (
	"bytes"
	"fmt"
	"github.com/GeoNet/fdsn/internal/fdsn"
	"github.com/GeoNet/kit/weft"
	"github.com/golang/groupcache"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"text/template"
)

const NO_DATA = 204

var (
	fdsnDataselectWadlFile []byte
	fdsnDataselectIndex    []byte
)

func init() {
	var err error
	var b bytes.Buffer

	t, err := template.New("t").ParseFiles("assets/tmpl/fdsn-ws-dataselect.wadl")
	if err != nil {
		log.Printf("error parsing assets/tmpl/fdsn-ws-dataselect.wadl: %s", err.Error())
	}
	err = t.ExecuteTemplate(&b, "body", os.Getenv("HOST_CNAME"))
	if err != nil {
		log.Printf("error executing assets/tmpl/fdsn-ws-dataselect.wadl: %s", err.Error())
	}
	fdsnDataselectWadlFile = b.Bytes()

	fdsnDataselectIndex, err = ioutil.ReadFile("assets/fdsn-ws-dataselect.html")
	if err != nil {
		log.Printf("error reading assets/fdsn-ws-dataselect.html: %s", err.Error())
	}
}

func fdsnDataselectV1Handler(r *http.Request, w http.ResponseWriter) (int64, error) {
	// the query parameters come from the URL or body.  This makes using weft.CheckQuery to complicated.
	// Additional work is done to check the method and parameters.

	var params []fdsn.DataSelect

	switch r.Method {
	case "POST":
		defer r.Body.Close()
		if err := fdsn.ParseDataSelectPost(r.Body, &params); err != nil {
			return 0, weft.StatusError{Code: http.StatusBadRequest, Err: err}
		}
		if len(params) == 0 {
			return 0, weft.StatusError{Code: NO_DATA, Err: fmt.Errorf("%s", "unable to parse post request")}
		}
	case "GET":
		d, err := fdsn.ParseDataSelectGet(r.URL.Query())
		if err != nil {
			return 0, weft.StatusError{Code: http.StatusBadRequest, Err: err}
		}

		params = append(params, d)
	default:
		return 0, weft.StatusError{Code: http.StatusMethodNotAllowed}
	}

	var keys []string
	var rec []byte

	// TODO - possibly limit request/response size and use a buffer for the response.  This
	// would make http response codes to the client more accurate.

	w.Header().Set("Content-Type", "application/vnd.fdsn.mseed")
	var n int
	var written int
	for _, v := range params {
		s, err := v.Regexp()
		if err != nil {
			return 0, err
		}
		keys, err = holdingsSearchNrt(s)
		if err != nil {
			return 0, err
		}
		for _, k := range keys {
			err = recordCache.Get(nil, k, groupcache.AllocatingByteSliceSink(&rec))
			switch err {
			case nil:
				n, err = w.Write(rec)
				if err != nil {
					return 0, err
				}
				written += n
			case errNoData:
			// do nothing for no data, it could be deleted from the db
			// before we get a chance to request it.
			default:
				return 0, err
			}
		}
	}

	if written == 0 {
		return 0, weft.StatusError{Code: params[0].NoData, Err: fmt.Errorf("%s", "no results for specified query")}
	}

	return int64(written), nil
}

func fdsnDataselectV1Index(r *http.Request, h http.Header, b *bytes.Buffer) error {
	err := weft.CheckQuery(r, []string{"GET"}, []string{}, []string{})
	if err != nil {
		return err
	}

	h.Set("Content-Type", "text/html; charset=utf-8")

	_, err = b.Write(fdsnDataselectIndex)

	return err
}

func fdsnDataselectVersion(r *http.Request, h http.Header, b *bytes.Buffer) error {
	err := weft.CheckQuery(r, []string{"GET"}, []string{}, []string{})
	if err != nil {
		return err
	}

	h.Set("Content-Type", "text/plain")
	_, err = b.WriteString("1.1")

	return err
}

func fdsnDataselectWadl(r *http.Request, h http.Header, b *bytes.Buffer) error {
	err := weft.CheckQuery(r, []string{"GET"}, []string{}, []string{})
	if err != nil {
		return err
	}

	h.Set("Content-Type", "application/xml")
	_, err = b.Write(fdsnDataselectWadlFile)

	return err
}
