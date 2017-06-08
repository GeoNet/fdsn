package main

import (
	"bytes"
	"github.com/GeoNet/fdsn/internal/fdsn"
	"github.com/GeoNet/weft"
	"github.com/golang/groupcache"
	"io/ioutil"
	"log"
	"net/http"
)

var (
	fdsnDataselectWadlFile []byte
	fdsnDataselectIndex    []byte
)

func init() {
	var err error
	fdsnDataselectWadlFile, err = ioutil.ReadFile("assets/fdsn-ws-dataselect.wadl")
	if err != nil {
		log.Printf("error reading assets/fdsn-ws-dataselect.wadl: %s", err.Error())
	}

	fdsnDataselectIndex, err = ioutil.ReadFile("assets/fdsn-ws-dataselect.html")
	if err != nil {
		log.Printf("error reading assets/fdsn-ws-dataselect.html: %s", err.Error())
	}
}

// fdsnDataselectV1Handler handles all dataselect queries.  It searches for matching keys in S3 and
// fetches them in parallel, writing matching records to w in the same order they were requested.
// This parses all input files before writing the StatusCode and before writing data to ResponseWriter.
// In the case of an error a non-200 status code is returned as a weft.Result and no output written to w.
func fdsnDataselectV1Handler(r *http.Request, w http.ResponseWriter) *weft.Result {
	var params []fdsn.DataSelect

	switch r.Method {
	case "POST":
		defer r.Body.Close()
		if err := fdsn.ParseDataSelectPost(r.Body, &params); err != nil {
			return weft.BadRequest(err.Error())
		}
	case "GET":
		d, err := fdsn.ParseDataSelectGet(r.URL.Query())
		if err != nil {
			return weft.BadRequest(err.Error())
		}

		params = append(params, d)
	default:
		return &weft.MethodNotAllowed
	}

	var err error
	var keys []string
	var rec []byte

	w.Header().Set("Content-Type", "application/vnd.fdsn.mseed")

	for _, v := range params {
		keys, err = holdingsSearchNrt(v.Regexp())
		if err != nil {
			return weft.InternalServerError(err)
		}
		for _, k := range keys {
			err = record.Get(nil, k, groupcache.AllocatingByteSliceSink(&rec))
			switch err {
			case nil:
				w.Write(rec)
			case errNoData:
			// do nothing for no data, it could be deleted from the db
			// before we get a chance to request it.
			default:
				return weft.InternalServerError(err)
			}
		}
	}

	return &weft.StatusOK
}

func fdsnDataselectV1Index(r *http.Request, h http.Header, b *bytes.Buffer) *weft.Result {
	switch r.Method {
	case "GET":
		if res := weft.CheckQuery(r, []string{}, []string{}); !res.Ok {
			return res
		}

		h.Set("Content-Type", "text/html")
		b.Write(fdsnDataselectIndex)
		return &weft.StatusOK
	default:
		return &weft.MethodNotAllowed
	}
}

func fdsnDataselectVersion(r *http.Request, h http.Header, b *bytes.Buffer) *weft.Result {
	switch r.Method {
	case "GET":
		if res := weft.CheckQuery(r, []string{}, []string{}); !res.Ok {
			return res
		}

		h.Set("Content-Type", "text/plain")
		b.WriteString("1.1")
		return &weft.StatusOK
	default:
		return &weft.MethodNotAllowed
	}
}

func fdsnDataselectWadl(r *http.Request, h http.Header, b *bytes.Buffer) *weft.Result {
	switch r.Method {
	case "GET":
		if res := weft.CheckQuery(r, []string{}, []string{}); !res.Ok {
			return res
		}

		h.Set("Content-Type", "application/xml")
		b.Write(fdsnDataselectWadlFile)
		return &weft.StatusOK
	default:
		return &weft.MethodNotAllowed
	}
}
