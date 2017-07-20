package main

import (
	"bytes"
	"github.com/GeoNet/fdsn/internal/fdsn"
	"github.com/GeoNet/fdsn/internal/weft"
	"github.com/golang/groupcache"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"text/template"
)

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

	// TODO - possibly limit request/response size and use a buffer for the response.  This
	// would make http response codes to the client more accurate.

	w.Header().Set("Content-Type", "application/vnd.fdsn.mseed")

	for _, v := range params {
		keys, err = holdingsSearchNrt(v.Regexp())
		if err != nil {
			return weft.InternalServerError(err)
		}
		for _, k := range keys {
			err = recordCache.Get(nil, k, groupcache.AllocatingByteSliceSink(&rec))
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
